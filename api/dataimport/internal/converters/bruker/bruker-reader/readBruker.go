package brukerreader

// Based on:
// https://github.com/hyperspy/hyperspy/blob/a676f9ad30c110e6e774e0f32136287c32c5ef69/hyperspy/io_plugins/bruker.py
// Converted by Claude sonnet 5 medium (free version) on 14 Sep 2026 to Go
// Some bug fixes later it appears functional

// Here follow the claude output comments:

// Package bruker is a Go translation of HyperSpy's Bruker BCF/SPX io_plugin
// (originally Python). It reads Bruker Esprit(R) ".bcf" hypermap containers
// (a proprietary "SFS" archive format holding 16-bit SEM images, EDS spectra,
// and an XML metadata tree) and ".spx" single-spectrum XML files.
//
// Notes on the port from Python:
//
//   - numpy arrays are represented as plain Go slices ([]uint8/16/32/64,
//     []int64, ...). There is no dask; "lazy" loading is modeled with a
//     LazyHyperCube func() (*HyperCube, error) closure instead of a dask
//     graph, so the caller decides when the (possibly expensive) hypermap
//     decode actually happens.
//   - ast.literal_eval is approximated by Interpret(), which recognizes
//     ints, floats, bools and "None"; anything else is left as a string.
//     This covers everything this format actually stores in attributes/text.
//   - xml.etree's dictionarize()/ClassInstance-flattening behavior is
//     reproduced with a generic XMLNode tree (see xmlnode.go) instead of
//     Go's static encoding/xml struct tags, since the metadata shape is
//     data-dependent.
//   - The Cython-accelerated "unbcf_fast" path from HyperSpy has no
//     equivalent here; only the pure-Python fallback path (py_parse_hypermap)
//     is translated, as pyParseHypermap.
//
// This file has been written and reviewed carefully but could not be
// compiled in this environment (no Go toolchain / network access to install
// one) -- please run `go build ./...` and `go vet ./...` before relying on
// it, especially around the bit-level EDS packet decoding.

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------
// Format registration metadata (mirrors the module-level constants in the
// original Python io_plugin).
// ---------------------------------------------------------------------

const (
	FormatName  = "Bruker"
	Description = `the proprietary format used by Bruker's
Esprit(R) software to save hypermaps together with 16bit SEM imagery,
EDS spectra and metadata describing the dimensions of the data and
SEM/TEM (limited) parameters`
	FullSupport = false
)

// FileExtensions recognized by this reader.
var FileExtensions = []string{"bcf", "spx"}

const DefaultExtension = 0

const (
	ReadsImages        = true
	ReadsSpectrum      = true
	ReadsSpectrumImage = true
	Writes             = false
	NonUniformAxis     = false
)

// Dictionarize reproduces the Python `dictionarize()` helper: it folds an
// XML subtree into a map[string]interface{}, collapsing repeated child tags
// into slices, moving attributes in (prefixed with "XmlClass" when the node
// also has children, exactly as upstream does), and unwrapping a
// "ClassInstance" tag transparently.
func Dictionarize(t *XMLNode) map[string]interface{} {
	if t == nil {
		return nil
	}
	var val interface{}
	hasAttrs := len(t.Attrs) > 0
	if hasAttrs {
		val = map[string]interface{}{}
	}
	children := t.Children
	if len(children) > 0 {
		grouped := map[string][]interface{}{}
		var order []string
		for _, c := range children {
			dc := Dictionarize(c)
			for k, v := range dc {
				if _, seen := grouped[k]; !seen {
					order = append(order, k)
				}
				grouped[k] = append(grouped[k], v)
			}
		}
		m := map[string]interface{}{}
		for _, k := range order {
			v := grouped[k]
			if len(v) == 1 {
				m[k] = Interpret(v[0])
			} else {
				m[k] = v
			}
		}
		val = m
	}
	if hasAttrs {
		m, ok := val.(map[string]interface{})
		if !ok {
			m = map[string]interface{}{}
		}
		for _, a := range t.Attrs {
			key := a.Name
			if len(children) > 0 {
				key = "XmlClass" + key
			}
			m[key] = Interpret(a.Value)
		}
		val = m
	}
	text := strings.TrimSpace(t.Text)
	if text != "" {
		if len(children) > 0 || hasAttrs {
			if m, ok := val.(map[string]interface{}); ok {
				m["#text"] = Interpret(text)
				val = m
			}
		} else {
			val = Interpret(text)
		}
	}
	d := map[string]interface{}{t.Tag: val}
	if ci, ok := d["ClassInstance"]; ok {
		if m, ok2 := ci.(map[string]interface{}); ok2 {
			return m
		}
		return map[string]interface{}{"#value": ci}
	}
	return d
}

func genIsoDateTime(node *XMLNode) (date, tstr string, ok bool) {
	if node == nil {
		return "", "", false
	}
	dateXML := node.Find("Date", "", "")
	timeXML := node.Find("Time", "", "")
	if dateXML == nil || timeXML == nil {
		return "", "", false
	}

	dateXMLStr := strings.TrimSpace(dateXML.Text)
	timeXMLStr := strings.TrimSpace(timeXML.Text)

	// If we have 0's missing, parsing fails
	bits := strings.Split(dateXMLStr, ".")

	if len(bits) == 3 {
		if len(bits[1]) == 1 {
			dateXMLStr = bits[0] + ".0" + bits[1] + "." + bits[2]
		}
		if len(bits[0]) == 1 {
			dateXMLStr = "0" + dateXMLStr
		}
	}

	bits = strings.Split(timeXMLStr, ":")

	if len(bits) == 3 {
		if len(bits[1]) == 1 {
			timeXMLStr = bits[0] + ":0" + bits[1] + ":" + bits[2]
		}
		if len(bits[0]) == 1 {
			timeXMLStr = "0" + timeXMLStr
		}
	}

	dt, err := time.Parse("02.01.2006 15:04:05", dateXMLStr+" "+timeXMLStr)
	if err != nil {
		return "", "", false
	}
	return dt.Format("2006-01-02"), dt.Format("15:04:05"), true
}

// ---------------------------------------------------------------------
// SFS container ("Scanning ... File System") reader.
// This is the binary archive format wrapping images/spectra/XML inside a
// .bcf file. It supports optional per-container zlib block compression.
// ---------------------------------------------------------------------

const (
	sfsMagic      = "AAMVHFSS"
	tableItemSize = 0x200 // 512 bytes per raw tree-item record
)

// SFSTreeItem is one entry (file or directory) in the SFS virtual file
// system tree.
type SFSTreeItem struct {
	sfs               *SFSReader
	PointerToPtrTable int32
	Size              uint64
	CreateTime        time.Time
	ModTime           time.Time
	SomeTime          time.Time
	Permissions       uint32
	Parent            int32
	IsDir             bool
	Name              string
	SizeInChunks      int
	Pointers          []int64 // absolute file offsets of each data chunk

	// zlib-compression metadata (only set when sfs.Compression == "zlib")
	uncompressedBlkSize uint32
	noOfComprBlk        uint32
}

func filetimeToUnix(ft uint64) time.Time {
	base := time.Date(1601, 1, 1, 0, 0, 0, 0, time.UTC)
	return base.Add(time.Duration(ft/10) * time.Microsecond)
}

func newSFSTreeItem(raw []byte, sfs *SFSReader) (*SFSTreeItem, error) {
	if len(raw) < tableItemSize {
		return nil, fmt.Errorf("bruker: tree item record too short (%d bytes)", len(raw))
	}
	le := binary.LittleEndian
	it := &SFSTreeItem{sfs: sfs}
	it.PointerToPtrTable = int32(le.Uint32(raw[0:4]))
	it.Size = le.Uint64(raw[4:12])
	createTime := le.Uint64(raw[12:20])
	modTime := le.Uint64(raw[20:28])
	someTime := le.Uint64(raw[28:36])
	it.CreateTime = filetimeToUnix(createTime)
	it.ModTime = filetimeToUnix(modTime)
	it.SomeTime = filetimeToUnix(someTime)
	it.Permissions = le.Uint32(raw[36:40])
	it.Parent = int32(le.Uint32(raw[40:44]))
	// raw[44:220] is reserved/unused (176 bytes)
	it.IsDir = raw[220] != 0
	// raw[221:224] is reserved/unused (3 bytes)
	nameBytes := raw[224:480]
	nul := bytes.IndexByte(nameBytes, 0)
	if nul >= 0 {
		nameBytes = nameBytes[:nul]
	}
	it.Name = string(nameBytes)
	// raw[480:512] is reserved/unused (32 bytes)

	it.SizeInChunks = ceilDiv64(it.Size, uint64(sfs.UsableChunk))
	if !it.IsDir {
		if err := it.fillPointerTable(); err != nil {
			return nil, err
		}
	}
	return it, nil
}

func ceilDiv64(a uint64, b uint64) int {
	if b == 0 {
		return 0
	}
	return int((a + b - 1) / b)
}

// fillPointerTable reads (possibly across several linked chunks) the array
// of chunk pointers for this item's data.
func (it *SFSTreeItem) fillPointerTable() error {
	entriesPerChunk := it.sfs.UsableChunk / 4
	nOfChunks := ceilDiv(it.SizeInChunks, int(entriesPerChunk))

	f, err := os.Open(it.sfs.Filename)
	if err != nil {
		return err
	}
	defer f.Close()

	var tempTable []byte
	if nOfChunks > 1 {
		nextChunk := int64(it.PointerToPtrTable)
		buf := &bytes.Buffer{}
		for i := 0; i < nOfChunks; i++ {
			if _, err := f.Seek(int64(it.sfs.ChunkSize)*nextChunk+0x118, io.SeekStart); err != nil {
				return err
			}
			var nc uint32
			if err := binary.Read(f, binary.LittleEndian, &nc); err != nil {
				return err
			}
			nextChunk = int64(nc)
			if _, err := f.Seek(28, io.SeekCurrent); err != nil {
				return err
			}
			chunk := make([]byte, it.sfs.UsableChunk)
			if _, err := io.ReadFull(f, chunk); err != nil {
				return err
			}
			buf.Write(chunk)
		}
		tempTable = buf.Bytes()
	} else {
		if _, err := f.Seek(int64(it.sfs.ChunkSize)*int64(it.PointerToPtrTable)+0x138, io.SeekStart); err != nil {
			return err
		}
		tempTable = make([]byte, it.sfs.UsableChunk)
		if _, err := io.ReadFull(f, tempTable); err != nil {
			return err
		}
	}

	n := it.SizeInChunks
	if len(tempTable) < n*4 {
		return fmt.Errorf("bruker: pointer table shorter than expected for %q", it.Name)
	}
	it.Pointers = make([]int64, n)
	for i := 0; i < n; i++ {
		v := binary.LittleEndian.Uint32(tempTable[i*4 : i*4+4])
		it.Pointers[i] = int64(v)*int64(it.sfs.ChunkSize) + 0x138
	}
	return nil
}

// ReadPiece reads `length` bytes starting at logical `offset` within the
// item's (uncompressed, chunk-scattered) data, following the pointer table
// across chunk boundaries as needed.
func (it *SFSTreeItem) ReadPiece(offset, length int64) ([]byte, error) {
	f, err := os.Open(it.sfs.Filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	usable := int64(it.sfs.UsableChunk)
	fbIdx := offset / usable
	fbo := offset % usable
	lbIdx := (offset + length) / usable
	lbco := (offset + length) % usable

	out := &bytes.Buffer{}
	if fbIdx != lbIdx {
		if _, err := f.Seek(it.Pointers[fbIdx]+fbo, io.SeekStart); err != nil {
			return nil, err
		}
		if _, err := io.CopyN(out, f, usable-fbo); err != nil {
			return nil, err
		}
		for i := fbIdx + 1; i < lbIdx; i++ {
			if _, err := f.Seek(it.Pointers[i], io.SeekStart); err != nil {
				return nil, err
			}
			if _, err := io.CopyN(out, f, usable); err != nil {
				return nil, err
			}
		}
		if lbco > 0 {
			if _, err := f.Seek(it.Pointers[lbIdx], io.SeekStart); err != nil {
				return nil, err
			}
			if _, err := io.CopyN(out, f, lbco); err != nil {
				return nil, err
			}
		}
	} else {
		if _, err := f.Seek(it.Pointers[fbIdx]+fbo, io.SeekStart); err != nil {
			return nil, err
		}
		if _, err := io.CopyN(out, f, length); err != nil {
			return nil, err
		}
	}
	return out.Bytes(), nil
}

// setupCompressionMetadata reads the per-item zlib block header (only
// called when the whole container uses zlib compression).
func (it *SFSTreeItem) setupCompressionMetadata() error {
	f, err := os.Open(it.sfs.Filename)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Seek(it.Pointers[0], io.SeekStart); err != nil {
		return err
	}
	hdr := make([]byte, 16)
	if _, err := io.ReadFull(f, hdr); err != nil {
		return err
	}
	aacs := binary.LittleEndian.Uint32(hdr[0:4])
	ucSize := binary.LittleEndian.Uint32(hdr[4:8])
	nBlocks := binary.LittleEndian.Uint32(hdr[12:16])
	if aacs != 0x53434141 { // "AACS" as little-endian uint32
		return errors.New("bruker: file marked compressed but AACS signature missing")
	}
	it.uncompressedBlkSize = ucSize
	it.noOfComprBlk = nBlocks
	return nil
}

// ChunkIterator streams an item's data one (decompressed, if applicable)
// chunk at a time. Next returns io.EOF once exhausted.
type ChunkIterator interface {
	Next() ([]byte, error)
}

type rawChunkIterator struct {
	it       *SFSTreeItem
	f        *os.File
	idx      int
	last     int
	lastSent bool
}

func (r *rawChunkIterator) Next() ([]byte, error) {
	if r.idx >= r.last-1 {
		if r.lastSent {
			return nil, io.EOF
		}
		r.lastSent = true
		if _, err := r.f.Seek(r.it.Pointers[r.last-1], io.SeekStart); err != nil {
			return nil, err
		}
		remainder := int(r.it.Size % uint64(r.it.sfs.UsableChunk))
		n := int(r.it.sfs.UsableChunk)
		if remainder != 0 {
			n = remainder
		}
		buf := make([]byte, n)
		if _, err := io.ReadFull(r.f, buf); err != nil {
			return nil, err
		}
		return buf, nil
	}
	if _, err := r.f.Seek(r.it.Pointers[r.idx], io.SeekStart); err != nil {
		return nil, err
	}
	buf := make([]byte, r.it.sfs.UsableChunk)
	if _, err := io.ReadFull(r.f, buf); err != nil {
		return nil, err
	}
	r.idx++
	return buf, nil
}

func (r *rawChunkIterator) Close() error { return r.f.Close() }

type comprChunkIterator struct {
	it     *SFSTreeItem
	idx    int
	count  int
	offset int64
}

func (c *comprChunkIterator) Next() ([]byte, error) {
	if c.idx >= c.count {
		return nil, io.EOF
	}
	hdr, err := c.it.ReadPiece(c.offset, 16)
	if err != nil {
		return nil, err
	}
	cprSize := int64(binary.LittleEndian.Uint32(hdr[0:4]))
	c.offset += 16
	raw, err := c.it.ReadPiece(c.offset, cprSize)
	if err != nil {
		return nil, err
	}
	c.offset += cprSize
	c.idx++
	zr, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return io.ReadAll(zr)
}

// GetIterAndProperties returns an iterator over this item's data plus the
// (blockSize, blockCount) pair, matching the Python method of the same
// purpose.
func (it *SFSTreeItem) GetIterAndProperties() (ChunkIterator, int, int, error) {
	switch it.sfs.Compression {
	case "None":
		f, err := os.Open(it.sfs.Filename)
		if err != nil {
			return nil, 0, 0, err
		}
		return &rawChunkIterator{it: it, f: f, last: it.SizeInChunks}, int(it.sfs.UsableChunk), it.SizeInChunks, nil
	case "zlib":
		return &comprChunkIterator{it: it, count: int(it.noOfComprBlk), offset: 0x80},
			int(it.uncompressedBlkSize), int(it.noOfComprBlk), nil
	default:
		return nil, 0, 0, fmt.Errorf("bruker: file %q is compressed with an unknown/unimplemented algorithm", it.sfs.Filename)
	}
}

// GetAsBytes reads the item's whole (decompressed) content into memory.
func (it *SFSTreeItem) GetAsBytes() ([]byte, error) {
	iter, _, _, err := it.GetIterAndProperties()
	if err != nil {
		return nil, err
	}
	if closer, ok := iter.(interface{ Close() error }); ok {
		defer closer.Close()
	}
	out := &bytes.Buffer{}
	for {
		chunk, err := iter.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		out.Write(chunk)
	}
	return out.Bytes(), nil
}

// SFSReader parses the SFS container header and virtual file-system tree.
type SFSReader struct {
	Filename     string
	ChunkSize    uint32
	UsableChunk  uint32
	SFSVersion   string
	TreeAddress  uint32
	NTreeItems   uint32
	SFSNOfChunks uint32
	Compression  string // "None" or "zlib"

	// VFS maps names to either map[string]interface{} (a directory) or
	// *SFSTreeItem (a file), mirroring the nested dict built by the
	// original Python `_flat_items_to_dict`.
	VFS map[string]interface{}

	items []*SFSTreeItem
}

func NewSFSReader(filename string) (*SFSReader, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	magic := make([]byte, 8)
	if _, err := io.ReadFull(f, magic); err != nil {
		return nil, err
	}
	if string(magic) != sfsMagic {
		return nil, fmt.Errorf("bruker: file %q is not an SFS container", filename)
	}

	s := &SFSReader{Filename: filename}

	if _, err := f.Seek(0x124, io.SeekStart); err != nil {
		return nil, err
	}
	var versionBits uint32
	if err := binary.Read(f, binary.LittleEndian, &versionBits); err != nil {
		return nil, err
	}
	version := math.Float32frombits(versionBits)
	if err := binary.Read(f, binary.LittleEndian, &s.ChunkSize); err != nil {
		return nil, err
	}
	s.SFSVersion = fmt.Sprintf("%4.2f", version)
	s.UsableChunk = s.ChunkSize - 32

	if _, err := f.Seek(0x140, io.SeekStart); err != nil {
		return nil, err
	}
	if err := binary.Read(f, binary.LittleEndian, &s.TreeAddress); err != nil {
		return nil, err
	}
	if err := binary.Read(f, binary.LittleEndian, &s.NTreeItems); err != nil {
		return nil, err
	}
	if err := binary.Read(f, binary.LittleEndian, &s.SFSNOfChunks); err != nil {
		return nil, err
	}

	if err := s.setupVFS(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *SFSReader) setupVFS() error {
	f, err := os.Open(s.Filename)
	if err != nil {
		return err
	}
	defer f.Close()

	nFileTreeChunks := ceilDiv(int(s.NTreeItems)*tableItemSize, int(s.ChunkSize)-0x20)

	var rawTree []byte
	if nFileTreeChunks == 1 {
		if _, err := f.Seek(int64(s.ChunkSize)*int64(s.TreeAddress)+0x138, io.SeekStart); err != nil {
			return err
		}
		rawTree = make([]byte, tableItemSize*int(s.NTreeItems))
		if _, err := io.ReadFull(f, rawTree); err != nil {
			return err
		}
	} else {
		buf := &bytes.Buffer{}
		treeAddress := int64(s.TreeAddress)
		itemsPerChunk := (int(s.ChunkSize) - 0x20) / tableItemSize
		for i := 0; i < nFileTreeChunks; i++ {
			if _, err := f.Seek(int64(s.ChunkSize)*treeAddress+0x118, io.SeekStart); err != nil {
				return err
			}
			var next uint32
			if err := binary.Read(f, binary.LittleEndian, &next); err != nil {
				return err
			}
			treeAddress = int64(next)
			if _, err := f.Seek(28, io.SeekCurrent); err != nil {
				return err
			}
			chunk := make([]byte, itemsPerChunk*tableItemSize)
			if _, err := io.ReadFull(f, chunk); err != nil {
				return err
			}
			buf.Write(chunk)
		}
		rawTree = buf.Bytes()[:int(s.NTreeItems)*tableItemSize]
	}

	items := make([]*SFSTreeItem, s.NTreeItems)
	for i := 0; i < int(s.NTreeItems); i++ {
		it, err := newSFSTreeItem(rawTree[i*tableItemSize:(i+1)*tableItemSize], s)
		if err != nil {
			return err
		}
		items[i] = it
	}
	s.items = items

	if err := s.checkCompression(items); err != nil {
		return err
	}
	if s.Compression == "zlib" {
		for _, it := range items {
			if !it.IsDir {
				if err := it.setupCompressionMetadata(); err != nil {
					return err
				}
			}
		}
	}

	s.VFS = flatItemsToDict(items)
	return nil
}

func (s *SFSReader) checkCompression(items []*SFSTreeItem) error {
	f, err := os.Open(s.Filename)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, it := range items {
		if it.IsDir {
			continue
		}
		if _, err := f.Seek(it.Pointers[0], io.SeekStart); err != nil {
			return err
		}
		sig := make([]byte, 4)
		if _, err := io.ReadFull(f, sig); err != nil {
			return err
		}
		if bytes.Equal(sig, []byte{0x41, 0x41, 0x43, 0x53}) { // "AACS"
			s.Compression = "zlib"
		} else {
			s.Compression = "None"
		}
		break
	}
	if s.Compression == "" {
		s.Compression = "None"
	}
	return nil
}

// flatItemsToDict reproduces `_flat_items_to_dict`: it resolves each item's
// full ancestor-name chain by walking Parent indices up to the sentinel -1
// (root), then builds a nested map keyed by those chains.
func flatItemsToDict(items []*SFSTreeItem) map[string]interface{} {
	n := len(items)
	paths := make([][]int, n)
	for i, it := range items {
		paths[i] = []int{int(it.Parent)}
	}
	allTerminal := func() bool {
		for _, p := range paths {
			if p[len(p)-1] != -1 {
				return false
			}
		}
		return true
	}
	for !allTerminal() {
		for f := 0; f < n; f++ {
			last := paths[f][len(paths[f])-1]
			if last != -1 {
				paths[f] = append(paths[f], paths[last]...)
			}
		}
	}

	names := make([]string, n+1)
	for i, it := range items {
		names[i] = it.Name
	}
	names[n] = "root" // sentinel for parent == -1

	pathNames := make([][]string, n)
	for i, p := range paths {
		seg := make([]string, len(p))
		for r, idx := range p {
			if idx == -1 {
				idx = n
			}
			seg[r] = names[idx]
		}
		// reverse in place
		for l, rr := 0, len(seg)-1; l < rr; l, rr = l+1, rr-1 {
			seg[l], seg[rr] = seg[rr], seg[l]
		}
		pathNames[i] = seg
	}

	root := map[string]interface{}{}
	for i, segs := range pathNames {
		dir := root
		for _, seg := range segs {
			next, ok := dir[seg]
			if !ok {
				m := map[string]interface{}{}
				dir[seg] = m
				dir = m
				continue
			}
			m, ok := next.(map[string]interface{})
			if !ok {
				m = map[string]interface{}{}
				dir[seg] = m
			}
			dir = m
		}
		if items[i].IsDir {
			dir[items[i].Name] = map[string]interface{}{}
		} else {
			dir[items[i].Name] = items[i]
		}
	}
	if r, ok := root["root"].(map[string]interface{}); ok {
		return r
	}
	return map[string]interface{}{}
}

// GetFile resolves a "/"-separated path within the VFS tree, returning
// either *SFSTreeItem (a file) or map[string]interface{} (a directory).
func (s *SFSReader) GetFile(path string) (interface{}, error) {
	var cur interface{} = s.VFS
	for _, part := range strings.Split(path, "/") {
		if part == "" {
			continue
		}
		m, ok := cur.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("bruker: %q is not a directory while resolving %q", part, path)
		}
		next, ok := m[part]
		if !ok {
			return nil, fmt.Errorf("bruker: path %q not found (missing %q)", path, part)
		}
		cur = next
	}
	return cur, nil
}

// ---------------------------------------------------------------------
// EDS spectrum metadata + counts
// ---------------------------------------------------------------------

type EDXSpectrum struct {
	HardwareMetadata map[string]interface{}
	Amplification    float64

	DetectorMetadata map[string]interface{}
	DetectorType     interface{}

	EsmaMetadata map[string]interface{}
	HV           float64 // primary beam energy, kV
	ElevAngle    float64

	Date, Time string

	SpectrumMetadata map[string]interface{}
	Offset           float64 // CalibAbs
	Scale            float64 // CalibLin

	Data []uint64 // raw channel counts
}

func NewEDXSpectrum(spectrum *XMLNode) (*EDXSpectrum, error) {
	trtHeader := spectrum.Find("TRTHeaderedClass", "", "")
	if trtHeader == nil {
		return nil, errors.New("bruker: spectrum missing TRTHeaderedClass")
	}
	hardwareHeader := trtHeader.Find("ClassInstance", "Type", "TRTSpectrumHardwareHeader")
	detectorHeader := trtHeader.Find("ClassInstance", "Type", "TRTDetectorHeader")
	esmaHeader := trtHeader.Find("ClassInstance", "Type", "TRTESMAHeader")
	spectrumHeader := spectrum.Find("ClassInstance", "Type", "TRTSpectrumHeader")
	xrfHeader := trtHeader.Find("ClassInstance", "Type", "TRTXrfHeader")

	es := &EDXSpectrum{}
	es.HardwareMetadata = Dictionarize(hardwareHeader)
	es.Amplification = asFloat(es.HardwareMetadata["Amplification"])

	es.DetectorMetadata = Dictionarize(detectorHeader)
	es.DetectorType = es.DetectorMetadata["Type"]

	if detLayersRaw, ok := es.DetectorMetadata["DetLayers"].(string); ok {
		decoded, err := base64.StdEncoding.DecodeString(detLayersRaw)
		if err == nil {
			zr, err := zlibNewReaderRaw(decoded)
			if err == nil {
				miniXML, err := parseXML(zr)
				if err == nil {
					layers := map[string]interface{}{}
					for _, c := range miniXML.Children {
						attrs := map[string]interface{}{}
						for _, a := range c.Attrs {
							attrs[a.Name] = a.Value
						}
						layers[c.Tag] = attrs
					}
					es.DetectorMetadata["DetLayers"] = layers
				}
			}
		}
	}

	if esmaHeader != nil {
		es.EsmaMetadata = Dictionarize(esmaHeader)
	}
	if xrfHeader != nil {
		xrfDict := Dictionarize(xrfHeader)
		es.EsmaMetadata = map[string]interface{}{
			"PrimaryEnergy":  xrfDict["Voltage"],
			"ElevationAngle": xrfDict["ExcitationAngle"],
		}
	}
	es.HV = asFloat(es.EsmaMetadata["PrimaryEnergy"])
	es.ElevAngle = asFloat(es.EsmaMetadata["ElevationAngle"])

	if date, tstr, ok := genIsoDateTime(spectrumHeader); ok {
		es.Date, es.Time = date, tstr
	}

	es.SpectrumMetadata = Dictionarize(spectrumHeader)
	es.Offset = asFloat(es.SpectrumMetadata["CalibAbs"])
	es.Scale = asFloat(es.SpectrumMetadata["CalibLin"])

	channelsNode := spectrum.Find("Channels", "", "")
	if channelsNode == nil {
		return nil, errors.New("bruker: spectrum missing Channels")
	}
	es.Data = parseUint64CSV(channelsNode.Text)

	return es, nil
}

// zlibNewReaderRaw fully decompresses a zlib-wrapped byte slice (this
// mirrors zlib.decompress() used directly on already-in-memory bytes, as
// opposed to the streaming ChunkIterator use above).
func zlibNewReaderRaw(data []byte) ([]byte, error) {
	zr, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return io.ReadAll(zr)
}

func parseUint64CSV(s string) []uint64 {
	parts := strings.Split(strings.TrimSpace(s), ",")
	out := make([]uint64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := strconv.ParseUint(p, 10, 64)
		if err == nil {
			out = append(out, v)
		}
	}
	return out
}

// LastNonZeroChannel returns the index of the last nonzero channel.
func (e *EDXSpectrum) LastNonZeroChannel() int {
	for i := len(e.Data) - 1; i >= 0; i-- {
		if e.Data[i] != 0 {
			return i
		}
	}
	return -1
}

// EnergyToChannel converts an energy (in kV, unless kV is false, in which
// case it is given in V) to a channel index.
func (e *EDXSpectrum) EnergyToChannel(energy float64, kV bool) int {
	enTemp := energy
	if !kV {
		enTemp = energy / 1000.0
	}
	return int(math.Round((enTemp - e.Offset) / e.Scale))
}

// ---------------------------------------------------------------------
// Images
// ---------------------------------------------------------------------

// Axis mirrors the small per-axis dict HyperSpy expects.
type Axis struct {
	Name   string
	Size   int
	Offset float64
	Scale  float64
	Units  string
}

// HspyItem mirrors the nested dict (data/axes/metadata/original_metadata)
// the Python plugin hands back to HyperSpy for each image/spectrum/hypermap.
type HspyItem struct {
	Data             interface{} // []uint8/[]uint16/[]uint32/[]float64/*HyperCube
	Axes             []Axis
	Metadata         map[string]interface{}
	OriginalMetadata map[string]interface{}
}

// BrukerImage holds one TRTImageData node's decoded planes.
type BrukerImage struct {
	Width, Height int
	Dtype         string // "u1", "u2", or "u4"
	PlaneCount    int
	Images        []*HspyItem
}

// planeAllZero reports whether every sample in a raw plane buffer is zero
// (mirrors Python's `if any(array1):` gate, inverted).
func planeAllZero(raw []byte) bool {
	for _, b := range raw {
		if b != 0 {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------
// HyperHeader: parses the big TRTSpectrumDatabase XML tree (SEM/stage
// metadata, images, element selections, and the per-index "sum" spectra).
// ---------------------------------------------------------------------

type ElementInfo struct {
	Line   string
	Energy float64
}

type HyperHeader struct {
	Name    string
	Date    string
	Time    string
	Version int

	SemMetadata   map[string]interface{}
	HV            float64
	Units         string // "µm" or "pix"
	XRes, YRes    float64
	StageMetadata map[string]interface{}
	DspMetadata   map[string]interface{}

	Mode string // "SEM" or "TEM"

	Image    *BrukerImage
	Overview *BrukerImage // optional

	Elements map[string]ElementInfo

	LineCounter  interface{}
	ChannelCount int
	MappingCount int

	SpectraData map[int]*EDXSpectrum
}

func NewHyperHeader(xmlBytes []byte, indexes []int, instrument string) (*HyperHeader, error) {
	doc, err := parseXML(xmlBytes)
	if err != nil {
		return nil, err
	}
	root := doc
	if doc.Tag != "ClassInstance" {
		root = doc.Find("ClassInstance", "Type", "TRTSpectrumDatabase")
	}
	if root == nil {
		root = doc.FindPath(PathStep{"ClassInstance", "Type", "TRTSpectrumDatabase"})
	}
	if root == nil {
		return nil, errors.New("bruker: could not find TRTSpectrumDatabase root")
	}

	h := &HyperHeader{}
	if name, ok := root.Attr("Name"); ok {
		h.Name = name
	} else {
		h.Name = "Undefinded"
	}

	hd := root.Find("Header", "", "")
	if hd == nil {
		return nil, errors.New("bruker: header missing Header node")
	}
	if date, tstr, ok := genIsoDateTime(hd); ok {
		h.Date, h.Time = date, tstr
	}
	if fv := hd.Find("FileVersion", "", ""); fv != nil {
		h.Version = asInt(Interpret(strings.TrimSpace(fv.Text)))
	}

	if err := h.setMicroscope(root); err != nil {
		return nil, err
	}
	h.setMode(instrument)
	if err := h.setImages(root); err != nil {
		return nil, err
	}
	h.Elements = map[string]ElementInfo{}
	h.setElements(root)

	if lc := root.Find("LineCounter", "", ""); lc != nil {
		h.LineCounter = Interpret(strings.TrimSpace(lc.Text))
	}
	if cc := root.Find("ChCount", "", ""); cc != nil {
		h.ChannelCount = asInt(Interpret(strings.TrimSpace(cc.Text)))
	}
	if dc := root.Find("DetectorCount", "", ""); dc != nil {
		h.MappingCount = asInt(Interpret(strings.TrimSpace(dc.Text)))
	}

	h.SpectraData = map[int]*EDXSpectrum{}
	if err := h.setSumEDX(root, indexes); err != nil {
		return nil, err
	}

	return h, nil
}

func (h *HyperHeader) setMicroscope(root *XMLNode) error {
	semData := root.Find("ClassInstance", "Type", "TRTSEMData")
	h.SemMetadata = Dictionarize(semData)
	h.HV = asFloat(h.SemMetadata["HV"])
	if _, ok := h.SemMetadata["DX"]; ok {
		h.Units = "\u00b5m"
	} else {
		h.Units = "pix"
	}
	h.XRes = 1.0
	h.YRes = 1.0
	if v, ok := h.SemMetadata["DX"]; ok {
		h.XRes = asFloat(v)
	}
	if v, ok := h.SemMetadata["DY"]; ok {
		h.YRes = asFloat(v)
	}

	stageData := root.Find("ClassInstance", "Type", "TRTSEMStageData")
	h.StageMetadata = Dictionarize(stageData)

	dsp := root.Find("ClassInstance", "Type", "TRTDSPConfiguration")
	h.DspMetadata = Dictionarize(dsp)
	return nil
}

func (h *HyperHeader) setMode(instrument string) {
	if instrument != "" {
		h.Mode = instrument
		return
	}
	h.Mode = guessMode(h.HV)
}

func guessMode(hv float64) string {
	if hv > 30.0 {
		return "TEM"
	}
	return "SEM"
}

// GetAcqInstrumentDict returns the "Acquisition_instrument" sub-dict
// HyperSpy expects, optionally including detector info for `index`.
func (h *HyperHeader) GetAcqInstrumentDict(detector bool, index int) map[string]interface{} {
	acq := map[string]interface{}{"beam_energy": h.HV}
	if mag, ok := h.SemMetadata["Mag"]; ok {
		acq["magnification"] = mag
	}
	if detector {
		spec := h.SpectraData[index]
		det := genDetectorNode(spec)
		det["EDS"].(map[string]interface{})["real_time"] = h.CalcRealTime()
		acq["Detector"] = det
	}
	return acq
}

func (h *HyperHeader) parseImage(node *XMLNode, overview bool) (*BrukerImage, error) {
	var overRect map[string]float64
	if overview {
		rectNode := node.FindPath(
			PathStep{"ChildClassInstances", "", ""},
		)
		var rect *XMLNode
		if rectNode != nil {
			for _, c := range rectNode.FindAll("ClassInstance", "Name", "Map") {
				rect = c
				break
			}
		}
		if rect != nil {
			solid := rect.Find("TRTSolidOverlayElement", "", "")
			if solid != nil {
				basic := solid.Find("TRTBasicLineOverlayElement", "", "")
				if basic != nil {
					overlayEl := basic.Find("TRTOverlayElement", "", "")
					if overlayEl != nil {
						d := Dictionarize(overlayEl)
						if rectField, ok := d["Rect"].(map[string]interface{}); ok {
							overRect = map[string]float64{
								"y1": asFloat(rectField["Top"]) * h.YRes,
								"x1": asFloat(rectField["Left"]) * h.XRes,
								"y2": asFloat(rectField["Bottom"]) * h.YRes,
								"x2": asFloat(rectField["Right"]) * h.XRes,
							}
						}
					}
				}
			}
		}
	}

	img := &BrukerImage{}
	widthN := node.Find("Width", "", "")
	heightN := node.Find("Height", "", "")
	itemSizeN := node.Find("ItemSize", "", "")
	planeCountN := node.Find("PlaneCount", "", "")
	if widthN == nil || heightN == nil || itemSizeN == nil || planeCountN == nil {
		return nil, errors.New("bruker: image node missing Width/Height/ItemSize/PlaneCount")
	}
	img.Width = asInt(Interpret(strings.TrimSpace(widthN.Text)))
	img.Height = asInt(Interpret(strings.TrimSpace(heightN.Text)))
	img.Dtype = "u" + strings.TrimSpace(itemSizeN.Text)
	img.PlaneCount = asInt(Interpret(strings.TrimSpace(planeCountN.Text)))

	for i := 0; i < img.PlaneCount; i++ {
		plane := node.Find(fmt.Sprintf("Plane%d", i), "", "")
		if plane == nil {
			continue
		}
		dataNode := plane.Find("Data", "", "")
		if dataNode == nil {
			continue
		}
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(dataNode.Text))
		if err != nil {
			return nil, err
		}
		if planeAllZero(raw) {
			continue
		}
		item := h.genHspyItemDictBasic()
		item.Data = decodeTypedBuffer(raw, img.Dtype)
		item.Axes[0].Size = img.Height
		item.Axes[1].Size = img.Width
		if item.Metadata["Signal"] == nil {
			item.Metadata["Signal"] = map[string]interface{}{}
		}
		item.Metadata["Signal"].(map[string]interface{})["record_by"] = "image"
		item.Metadata["General"] = map[string]interface{}{}
		if desc := plane.Find("Description", "", ""); desc != nil {
			item.Metadata["General"].(map[string]interface{})["title"] = strings.TrimSpace(desc.Text)
		}
		if overview && overRect != nil {
			item.Metadata["Markers"] = map[string]interface{}{
				"overview": map[string]interface{}{
					"marker_type":    "Rectangle",
					"plot_on_signal": true,
					"data":           overRect,
					"marker_properties": map[string]interface{}{
						"color":     "yellow",
						"linewidth": 2,
					},
				},
			}
		}
		img.Images = append(img.Images, item)
	}
	return img, nil
}

// decodeTypedBuffer reinterprets raw little-endian bytes according to a
// numpy-style dtype string ("u1"/"u2"/"u4").
func decodeTypedBuffer(raw []byte, dtype string) interface{} {
	switch dtype {
	case "u1":
		return append([]byte(nil), raw...)
	case "u2":
		n := len(raw) / 2
		out := make([]uint16, n)
		for i := 0; i < n; i++ {
			out[i] = binary.LittleEndian.Uint16(raw[i*2 : i*2+2])
		}
		return out
	case "u4":
		n := len(raw) / 4
		out := make([]uint32, n)
		for i := 0; i < n; i++ {
			out[i] = binary.LittleEndian.Uint32(raw[i*4 : i*4+4])
		}
		return out
	default:
		return raw
	}
}

func (h *HyperHeader) setImages(root *XMLNode) error {
	imageNodes := root.FindAll("ClassInstance", "Type", "TRTImageData")
	var imageNode *XMLNode
	for _, n := range imageNodes {
		if _, hasName := n.Attr("Name"); !hasName {
			imageNode = n
		}
	}
	if imageNode == nil {
		return errors.New("bruker: could not find the main (unnamed) TRTImageData node")
	}
	img, err := h.parseImage(imageNode, false)
	if err != nil {
		return err
	}
	h.Image = img

	if h.Version == 2 {
		container := root.Find("ClassInstance", "Type", "TRTContainerClass")
		if container != nil {
			childInsts := container.Find("ChildClassInstances", "", "")
			if childInsts != nil {
				var overviewContainer *XMLNode
				for _, c := range childInsts.FindAll("ClassInstance", "Name", "OverviewImages") {
					overviewContainer = c
					break
				}
				if overviewContainer != nil {
					innerChild := overviewContainer.Find("ChildClassInstances", "", "")
					if innerChild != nil {
						overviewNodes := innerChild.FindAll("ClassInstance", "Type", "TRTImageData")
						if len(overviewNodes) > 0 {
							ov, err := h.parseImage(overviewNodes[0], true)
							if err != nil {
								return err
							}
							h.Overview = ov
						}
					}
				}
			}
		}
	}
	return nil
}

func (h *HyperHeader) setElements(root *XMLNode) {
	container := root.Find("ClassInstance", "Type", "TRTContainerClass")
	if container == nil {
		return
	}
	childInsts := container.Find("ChildClassInstances", "", "")
	if childInsts == nil {
		return
	}
	elemInfoList := childInsts.Find("ClassInstance", "Type", "TRTElementInformationList")
	if elemInfoList == nil {
		return
	}
	regionList := elemInfoList.Find("ClassInstance", "Type", "TRTSpectrumRegionList")
	if regionList == nil {
		return
	}
	elements := regionList.Find("ChildClassInstances", "", "")
	if elements == nil {
		return
	}
	for _, j := range elements.FindAll("ClassInstance", "Type", "TRTSpectrumRegion") {
		d := Dictionarize(j)
		name := asString(d["XmlClassName"])
		h.Elements[name] = ElementInfo{
			Line:   asString(d["Line"]),
			Energy: asFloat(d["Energy"]),
		}
	}
}

func (h *HyperHeader) setSumEDX(root *XMLNode, indexes []int) error {
	for _, i := range indexes {
		specNode := root.FindPath(PathStep{fmt.Sprintf("SpectrumData%d", i), "", ""})
		if specNode == nil {
			return fmt.Errorf("bruker: missing SpectrumData%d", i)
		}
		classInst := specNode.Find("ClassInstance", "", "")
		if classInst == nil {
			return fmt.Errorf("bruker: SpectrumData%d missing ClassInstance", i)
		}
		spec, err := NewEDXSpectrum(classInst)
		if err != nil {
			return err
		}
		h.SpectraData[i] = spec
	}
	return nil
}

// EstimateMapChannels mirrors HyperHeader.estimate_map_channels.
func (h *HyperHeader) EstimateMapChannels(index int) int {
	spec := h.SpectraData[index]
	brukerHVRange := spec.Amplification / 1000
	if h.HV >= brukerHVRange {
		return len(spec.Data)
	}
	return spec.EnergyToChannel(h.HV, true)
}

// EstimateMapDepth mirrors HyperHeader.estimate_map_depth, returning the
// smallest unsigned (or, if forNumpy, possibly signed) integer width in
// bytes needed to hold per-pixel channel sums.
func (h *HyperHeader) EstimateMapDepth(index, downsample int, forNumpy bool) (bitWidth int, signed bool) {
	sumEDS := h.SpectraData[index].Data
	var maxVal uint64
	for _, v := range sumEDS {
		if v > maxVal {
			maxVal = v
		}
	}
	roof := maxVal / uint64(h.Image.Width) / uint64(h.Image.Height) * 2 * uint64(downsample) * uint64(downsample)

	if roof > 0xFF {
		if roof > 0xFFFF {
			if forNumpy && downsample > 1 {
				if roof > 0xEFFFFFFF {
					return 64, true
				}
				return 32, true
			}
			return 32, false
		}
		if forNumpy && downsample > 1 {
			if roof > 0xEFFF {
				return 32, true
			}
			return 16, true
		}
		return 16, false
	}
	if forNumpy && downsample > 1 {
		if roof > 0xEF {
			return 16, true
		}
		return 8, true
	}
	return 8, false
}

func (h *HyperHeader) GetSpectraMetadata(index int) *EDXSpectrum { return h.SpectraData[index] }

// CalcRealTime mirrors HyperHeader.calc_real_time (seconds).
func (h *HyperHeader) CalcRealTime() float64 {
	var lineCntSum float64
	switch lc := h.LineCounter.(type) {
	case []interface{}:
		for _, v := range lc {
			lineCntSum += asFloat(v)
		}
	default:
		lineCntSum = asFloat(h.LineCounter)
	}
	lineAvg := asFloat(h.DspMetadata["LineAverage"])
	pixAvg := asFloat(h.DspMetadata["PixelAverage"])
	pixTime := asFloat(h.DspMetadata["PixelTime"])
	width := float64(h.Image.Width)
	return lineCntSum * lineAvg * pixAvg * pixTime * width * 1e-6
}

func (h *HyperHeader) genHspyItemDictBasic() *HspyItem {
	return &HspyItem{
		Axes: []Axis{
			{Name: "height", Offset: 0, Scale: h.YRes, Units: h.Units},
			{Name: "width", Offset: 0, Scale: h.XRes, Units: h.Units},
		},
		Metadata: map[string]interface{}{
			"Acquisition_instrument": map[string]interface{}{
				h.Mode: h.GetAcqInstrumentDict(false, 0),
			},
			"Sample": map[string]interface{}{"name": h.Name},
		},
		OriginalMetadata: map[string]interface{}{
			"Microscope":        h.SemMetadata,
			"DSP Configuration": h.DspMetadata,
			"Stage":             h.StageMetadata,
		},
	}
}

func genDetectorNode(spec *EDXSpectrum) map[string]interface{} {
	eds := map[string]interface{}{
		"elevation_angle": spec.ElevAngle,
		"detector_type":   spec.DetectorType,
	}
	if v, ok := spec.EsmaMetadata["AzimutAngle"]; ok {
		eds["azimuth_angle"] = v
	}
	if v, ok := spec.HardwareMetadata["RealTime"]; ok {
		eds["real_time"] = asFloat(v) / 1000
		eds["live_time"] = asFloat(spec.HardwareMetadata["LifeTime"]) / 1000
	}
	return map[string]interface{}{"EDS": eds}
}

// ---------------------------------------------------------------------
// BCFReader: ties an SFSReader + HyperHeader together and exposes hypermap
// parsing.
// ---------------------------------------------------------------------

type BCFReader struct {
	*SFSReader
	Header           *HyperHeader
	AvailableIndexes []int
	DefIndex         int
}

func NewBCFReader(filename string, instrument string) (*BCFReader, error) {
	sfs, err := NewSFSReader(filename)
	if err != nil {
		return nil, err
	}
	headerFileRaw, err := sfs.GetFile("EDSDatabase/HeaderData")
	if err != nil {
		return nil, err
	}
	headerFile, ok := headerFileRaw.(*SFSTreeItem)
	if !ok {
		return nil, errors.New("bruker: EDSDatabase/HeaderData is not a file")
	}

	edsDBRaw, err := sfs.GetFile("EDSDatabase")
	if err != nil {
		return nil, err
	}
	edsDB, ok := edsDBRaw.(map[string]interface{})
	if !ok {
		return nil, errors.New("bruker: EDSDatabase is not a directory")
	}

	var indexes []int
	for name := range edsDB {
		if strings.Contains(name, "SpectrumData") {
			last := name[len(name)-1:]
			if idx, err := strconv.Atoi(last); err == nil {
				indexes = append(indexes, idx)
			}
		}
	}
	sort.Ints(indexes)
	if len(indexes) == 0 {
		return nil, errors.New("bruker: no SpectrumData indexes found")
	}

	headerBytes, err := headerFile.GetAsBytes()
	if err != nil {
		return nil, err
	}
	headerBytes = fixDecCommas(headerBytes)

	header, err := NewHyperHeader(headerBytes, indexes, instrument)
	if err != nil {
		return nil, err
	}

	return &BCFReader{
		SFSReader:        sfs,
		Header:           header,
		AvailableIndexes: indexes,
		DefIndex:         indexes[0],
	}, nil
}

func (b *BCFReader) CheckIndexValid(index int) (int, error) {
	for _, i := range b.AvailableIndexes {
		if i == index {
			return index, nil
		}
	}
	return 0, fmt.Errorf("bruker: requested index %d not in available indexes %v", index, b.AvailableIndexes)
}

// HyperCube holds a decoded 3D (height, width, channel) hypermap as a flat,
// row-major slice of 64-bit signed accumulators (wide enough for any of the
// dtypes the original numpy-based decoder would pick).
type HyperCube struct {
	Data  []HypermapValues
	Shape [3]int // height, width, channels
}

func (c *HyperCube) At(y, x, ch int) HypermapValues {
	return c.Data[(y*c.Shape[1]+x)*c.Shape[2]+ch]
}

// LazyHyperCube defers the (potentially expensive) hypermap decode until
// invoked, standing in for the dask.delayed/dask.array behavior upstream.
type LazyHyperCube func() (*HyperCube, error)

// CutoffMode selects how parse_hypermap picks the maximum channel to keep.
type CutoffMode int

const (
	CutoffNone CutoffMode = iota
	CutoffFixedKV
	CutoffAuto
	CutoffLowestConstraint
)

type ParseHypermapOptions struct {
	Index      int // <=0 means "use header.DefIndex"
	HasIndex   bool
	Downsample int // default 1
	Cutoff     CutoffMode
	CutoffKV   float64
}

// Issues found:
// b.header.date/time are empty
// Runs out of memory when allocating array of int64s to store all the shape^3 data

// ParseHypermap decodes one SpectrumData index into a HyperCube (eager) or
// a LazyHyperCube (deferred), mirroring BCF_reader.parse_hypermap.
func (b *BCFReader) ParseHypermap(opts ParseHypermapOptions, lazy bool) (*HyperCube, LazyHyperCube, error) {
	index := b.DefIndex
	if opts.HasIndex {
		index = opts.Index
	}
	downsample := opts.Downsample
	if downsample <= 0 {
		downsample = 1
	}

	eds := b.Header.SpectraData[index]
	var maxChan int
	switch opts.Cutoff {
	case CutoffFixedKV:
		maxChan = eds.EnergyToChannel(opts.CutoffKV, true)
	case CutoffAuto:
		maxChan = eds.LastNonZeroChannel()
	case CutoffLowestConstraint:
		maxChan = b.Header.EstimateMapChannels(index)
	default:
		maxChan = len(eds.Data) + 1
	}

	shape := [3]int{
		ceilDiv(b.Header.Image.Height, downsample),
		ceilDiv(b.Header.Image.Width, downsample),
		maxChan,
	}

	decode := func() (*HyperCube, error) {
		// Reopen the SFS container so concurrent decodes (e.g. one per
		// goroutine, standing in for dask's parallelism) don't share a
		// single *os.File.
		sfs, err := NewSFSReader(b.Filename)
		if err != nil {
			return nil, err
		}
		fileRaw, err := sfs.GetFile(fmt.Sprintf("EDSDatabase/SpectrumData%d", index))
		if err != nil {
			return nil, err
		}
		item, ok := fileRaw.(*SFSTreeItem)
		if !ok {
			return nil, fmt.Errorf("bruker: SpectrumData%d is not a file", index)
		}
		iter, blockSize, _, err := item.GetIterAndProperties()
		if err != nil {
			return nil, err
		}
		if closer, ok := iter.(interface{ Close() error }); ok {
			defer closer.Close()
		}
		flat, err := pyParseHypermap(iter, blockSize, shape, downsample)
		if err != nil {
			return nil, err
		}
		return &HyperCube{Data: flat, Shape: shape}, nil
	}

	if lazy {
		return nil, LazyHyperCube(decode), nil
	}
	cube, err := decode()
	return cube, nil, err
}

func (b *BCFReader) AddFilenameToGeneral(item *HspyItem) {
	if item.Metadata["General"] == nil {
		item.Metadata["General"] = map[string]interface{}{}
	}
	item.Metadata["General"].(map[string]interface{})["original_filename"] = filepath.Base(b.Filename)
}

// ---------------------------------------------------------------------
// py_parse_hypermap: the pure-Go port of HyperSpy's pure-Python fallback
// bit-level hypermap decoder (no Cython "unbcf_fast" equivalent here).
// ---------------------------------------------------------------------

// stFmtWidth mirrors the Python `st = {1: 'B', 2: 'B', 4: 'H', 8: 'I', 16: 'Q'}`
// lookup: given size_p (the byte-width of the packed "gain" field, one of
// 1/2/4/8), returns the byte-width of each element in the pixel/channel
// array that follows (size_p/2, per the original struct-format mapping).
func stFmtElemWidth(sizeP int) int { return sizeP / 2 }

func readUintLE(b []byte, width int) uint64 {
	var v uint64
	for i := width - 1; i >= 0; i-- {
		v = v<<8 | uint64(b[i])
	}
	return v
}

func bincountUint(vals []uint32, minlength int) []int64 {
	maxV := minlength - 1
	for _, v := range vals {
		if int(v) > maxV {
			maxV = int(v)
		}
	}
	if maxV < 0 {
		maxV = 0
	}
	out := make([]int64, maxV+1)
	for _, v := range vals {
		out[v]++
	}
	if len(out) < minlength {
		grown := make([]int64, minlength)
		copy(grown, out)
		out = grown
	}
	return out
}

// getMore drops the already-consumed prefix of buffer (everything before
// offset), appends the next chunk from iter, and returns the new buffer
// together with the reset offset (0), matching:
//
//	buffer1 = buffer1[offset:] + next(iter_data)
//	offset = 0
func getMore(buffer []byte, offset int, iter ChunkIterator) ([]byte, int, error) {
	next, err := iter.Next()
	if err != nil {
		return nil, 0, err
	}
	out := make([]byte, 0, len(buffer)-offset+len(next))
	out = append(out, buffer[offset:]...)
	out = append(out, next...)
	return out, 0, nil
}

type HypermapValues uint16

func pyParseHypermap(iterData ChunkIterator, sizeChnk int, shape [3]int, downsample int) ([]HypermapValues, error) {
	dwnFactor := downsample
	maxChan := shape[2]

	buffer1, err := iterData.Next()
	if err != nil {
		return nil, err
	}
	if len(buffer1) < 8 {
		return nil, errors.New("bruker: hypermap stream truncated before header")
	}
	height := int(int32(binary.LittleEndian.Uint32(buffer1[0:4])))
	width := int(int32(binary.LittleEndian.Uint32(buffer1[4:8])))

	vfa := make([]HypermapValues, shape[0]*shape[1]*shape[2])

	offset := 0x1A0
	size := sizeChnk

	for lineCnt := 0; lineCnt < height; lineCnt++ {
		if offset+4 >= size {
			newBuf, newOff, err := getMore(buffer1, offset, iterData)
			if err != nil {
				return nil, err
			}
			size = sizeChnk + size - offset
			buffer1, offset = newBuf, newOff
		}
		lineHead := int(int32(binary.LittleEndian.Uint32(buffer1[offset : offset+4])))
		offset += 4

		for p := 0; p < lineHead; p++ {
			if offset+22 >= size {
				newBuf, newOff, err := getMore(buffer1, offset, iterData)
				if err != nil {
					return nil, err
				}
				size = sizeChnk + size - offset
				buffer1, offset = newBuf, newOff
			}
			// '<IHHIHHHI': xPix(u32) chan1(u16) chan2(u16) dummy1(u32) flag(u16)
			//              dummySize1(u16) nOfPulses(u16) dataSize2(u32)
			hdr := buffer1[offset : offset+22]
			xPix := binary.LittleEndian.Uint32(hdr[0:4])
			chan1 := int(binary.LittleEndian.Uint16(hdr[4:6]))
			chan2 := int(binary.LittleEndian.Uint16(hdr[6:8]))
			// dummy1 (hdr[8:12]) unused
			flag := binary.LittleEndian.Uint16(hdr[12:14])
			// dummySize1 (hdr[14:16]) unused
			nOfPulses := int(binary.LittleEndian.Uint16(hdr[16:18]))
			dataSize2 := int(binary.LittleEndian.Uint32(hdr[18:22]))

			pixIdx := int(xPix)/dwnFactor + ceilDiv(width, dwnFactor)*(lineCnt/dwnFactor)
			offset += 22

			if offset+dataSize2 >= size {
				newBuf, newOff, err := getMore(buffer1, offset, iterData)
				if err != nil {
					return nil, err
				}
				size = sizeChnk + size - offset
				buffer1, offset = newBuf, newOff
			}

			var pixel []int64

			switch {
			case flag == 0:
				data1 := buffer1[offset : offset+dataSize2]
				n := len(data1) / 2
				vals := make([]uint32, n)
				for i := 0; i < n; i++ {
					vals[i] = uint32(binary.LittleEndian.Uint16(data1[i*2 : i*2+2]))
				}
				pixel = bincountUint(vals, chan1-1)
				offset += dataSize2

			case flag == 1:
				data1 := append([]byte(nil), buffer1[offset:offset+dataSize2]...)
				// byteswap every uint16 word in place
				for i := 0; i+1 < len(data1); i += 2 {
					data1[i], data1[i+1] = data1[i+1], data1[i]
				}
				// repeat each byte twice
				data2 := make([]byte, len(data1)*2)
				for i, b := range data1 {
					data2[2*i] = b
					data2[2*i+1] = b
				}
				// mask: drop indices where (i % 6) is 0 or 5
				selected := make([]byte, 0, len(data2))
				for i, b := range data2 {
					if m := i % 6; m == 0 || m == 5 {
						continue
					}
					selected = append(selected, b)
				}
				need := nOfPulses * 2
				if need > len(selected) {
					need = len(selected) - (len(selected) % 2)
				}
				exp16 := make([]uint32, need/2)
				for i := 0; i < len(exp16); i++ {
					v := binary.BigEndian.Uint16(selected[i*2 : i*2+2])
					if i%2 == 0 {
						v >>= 4
					}
					v &= 0x0FFF
					exp16[i] = uint32(v)
				}
				pixel = bincountUint(exp16, chan1-1)
				offset += dataSize2

			default: // flag > 1: variable-length instructive packing
				pixelI := make([]int64, 0, chan1)
				theEnd := offset + dataSize2 - 4
				for offset < theEnd {
					sizeP := int(buffer1[offset])
					channels := int(buffer1[offset+1])
					offset += 2
					if sizeP == 0 {
						for k := 0; k < channels; k++ {
							pixelI = append(pixelI, 0)
						}
						continue
					}
					gain := int64(readUintLE(buffer1[offset:offset+sizeP], sizeP))
					offset += sizeP
					if sizeP == 1 {
						length := ceilDiv(channels, 2)
						a := buffer1[offset : offset+length]
						g := make([]int64, 0, length*2)
						for _, byteVal := range a {
							g = append(g, int64(byteVal&0x0F)+gain, int64(byteVal>>4)+gain)
						}
						if len(g) > channels {
							g = g[:channels]
						}
						pixelI = append(pixelI, g...)
						offset += length
					} else {
						elemWidth := stFmtElemWidth(sizeP)
						length := channels * sizeP / 2
						for k := 0; k < channels; k++ {
							v := int64(readUintLE(buffer1[offset+k*elemWidth:offset+(k+1)*elemWidth], elemWidth))
							pixelI = append(pixelI, v+gain)
						}
						offset += length
					}
				}
				if chan2 < chan1 {
					rest := chan1 - chan2
					for k := 0; k < rest; k++ {
						pixelI = append(pixelI, 0)
					}
				}
				if nOfPulses > 0 {
					addS := int(binary.LittleEndian.Uint32(buffer1[offset : offset+4]))
					offset += 4
					if offset+addS >= size {
						newBuf, newOff, err := getMore(buffer1, offset, iterData)
						if err != nil {
							return nil, err
						}
						size = sizeChnk + size - offset
						buffer1, offset = newBuf, newOff
					}
					addPulses := make([]uint16, addS/2)
					for i := range addPulses {
						addPulses[i] = binary.LittleEndian.Uint16(buffer1[offset+i*2 : offset+i*2+2])
					}
					offset += addS
					for _, idx := range addPulses {
						if int(idx) < len(pixelI) {
							pixelI[idx]++
						}
					}
				} else {
					offset += 4
				}
				pixel = pixelI
			}

			effChan1 := chan1
			if maxChan < chan1 {
				effChan1 = maxChan
			}
			base := maxChan * pixIdx
			n := effChan1
			if n > len(pixel) {
				n = len(pixel)
			}
			if dwnFactor == 1 {
				for k := 0; k < n; k++ {
					vfa[base+k] = HypermapValues(pixel[k])
				}
			} else {
				for k := 0; k < n; k++ {
					vfa[base+k] += HypermapValues(pixel[k])
				}
			}
		}
	}

	return vfa, nil
}

// ---------------------------------------------------------------------
// SPX (single-spectrum XML) reader
// ---------------------------------------------------------------------

func SpxReader(filename string) (*HspyItem, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	root, err := parseXML(data)
	if err != nil {
		return nil, err
	}
	spNode := root
	if root.Tag != "ClassInstance" {
		spNode = root.Find("ClassInstance", "Type", "TRTSpectrum")
	}
	if spNode == nil {
		return nil, errors.New("bruker: could not find TRTSpectrum node")
	}
	name := "Undefinded"
	if n, ok := spNode.Attr("Name"); ok {
		name = n
	}

	spectrum, err := NewEDXSpectrum(spNode)
	if err != nil {
		return nil, err
	}
	mode := guessMode(spectrum.HV)

	resultsXML := spNode.Find("ClassInstance", "Type", "TRTResult")
	elementsXML := spNode.Find("ClassInstance", "Type", "TRTPSEElementList")

	metadata := map[string]interface{}{
		"Acquisition_instrument": map[string]interface{}{
			mode: map[string]interface{}{
				"Detector":    genDetectorNode(spectrum),
				"beam_energy": spectrum.HV,
			},
		},
		"General": map[string]interface{}{
			"original_filename": filepath.Base(filename),
			"title":             "EDX",
			"date":              spectrum.Date,
			"time":              spectrum.Time,
		},
		"Sample": map[string]interface{}{"name": name},
		"Signal": map[string]interface{}{
			"signal_type": "EDS_" + mode,
			"record_by":   "spectrum",
			"quantity":    "X-rays (Counts)",
		},
	}

	originalMetadata := map[string]interface{}{
		"Hardware": spectrum.HardwareMetadata,
		"Detector": spectrum.DetectorMetadata,
		"Analysis": spectrum.EsmaMetadata,
		"Spectrum": spectrum.SpectrumMetadata,
	}
	if resultsXML != nil {
		originalMetadata["Results"] = Dictionarize(resultsXML)
	}
	if elementsXML != nil {
		elemDict := Dictionarize(elementsXML)
		if childInsts, ok := elemDict["ChildClassInstances"]; ok {
			originalMetadata["Selected_elements"] = childInsts
			if m, ok := childInsts.(map[string]interface{}); ok {
				metadata["Sample"].(map[string]interface{})["elements"] = m["XmlClassName"]
			}
		}
	}

	data64 := make([]float64, len(spectrum.Data))
	for i, v := range spectrum.Data {
		data64[i] = float64(v)
	}

	return &HspyItem{
		Data: data64,
		Axes: []Axis{
			{Name: "Energy", Size: len(spectrum.Data), Offset: spectrum.Offset, Scale: spectrum.Scale, Units: "keV"},
		},
		Metadata:         metadata,
		OriginalMetadata: originalMetadata,
	}, nil
}

// ---------------------------------------------------------------------
// Top-level dispatch + BCF images/hyperspectra assembly
// ---------------------------------------------------------------------

type SelectType int

const (
	SelectAll SelectType = iota
	SelectImage
	SelectSpectrumImage
)

// FileReader dispatches on extension, mirroring the Python `file_reader`.
func FileReader(filename string, selectType SelectType, index int, hasIndex bool,
	downsample int, cutoff CutoffMode, cutoffKV float64, instrument string, lazy bool) ([]*HspyItem, error) {

	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
	switch ext {
	case "bcf":
		return BcfReader(filename, selectType, index, hasIndex, downsample, cutoff, cutoffKV, instrument, lazy)
	case "spx":
		item, err := SpxReader(filename)
		if err != nil {
			return nil, err
		}
		return []*HspyItem{item}, nil
	default:
		return nil, fmt.Errorf("bruker: unrecognized extension %q", ext)
	}
}

func BcfReader(filename string, selectType SelectType, index int, hasIndex bool,
	downsample int, cutoff CutoffMode, cutoffKV float64, instrument string, lazy bool) ([]*HspyItem, error) {

	bcf, err := NewBCFReader(filename, instrument)
	if err != nil {
		return nil, err
	}

	switch selectType {
	case SelectImage:
		return BcfImages(bcf), nil
	case SelectSpectrumImage:
		return BcfHyperspectra(bcf, index, hasIndex, downsample, cutoff, cutoffKV, lazy)
	default:
		imgs := BcfImages(bcf)
		specs, err := BcfHyperspectra(bcf, index, hasIndex, downsample, cutoff, cutoffKV, lazy)
		if err != nil {
			return nil, err
		}
		return append(imgs, specs...), nil
	}
}

func BcfImages(bcf *BCFReader) []*HspyItem {
	var out []*HspyItem
	for _, img := range bcf.Header.Image.Images {
		bcf.AddFilenameToGeneral(img)
		out = append(out, img)
	}
	if bcf.Header.Overview != nil {
		for _, img := range bcf.Header.Overview.Images {
			bcf.AddFilenameToGeneral(img)
			out = append(out, img)
		}
	}
	return out
}

// BcfHyperspectra decodes one or more SpectrumData indexes into HspyItems.
// When lazy is true, item.Data is a LazyHyperCube instead of a *HyperCube.
func BcfHyperspectra(bcf *BCFReader, index int, hasIndex bool, downsample int,
	cutoff CutoffMode, cutoffKV float64, lazy bool) ([]*HspyItem, error) {

	if downsample <= 0 {
		downsample = 1
	}

	var indexes []int
	if !hasIndex {
		indexes = []int{bcf.DefIndex}
	} else if index == -1 { // sentinel for python's index='all'
		indexes = bcf.AvailableIndexes
	} else {
		valid, err := bcf.CheckIndexValid(index)
		if err != nil {
			return nil, err
		}
		indexes = []int{valid}
	}

	mode := bcf.Header.Mode
	mapping := getMapping(mode)

	var out []*HspyItem
	for _, idx := range indexes {
		cube, lazyCube, err := bcf.ParseHypermap(ParseHypermapOptions{
			Index: idx, HasIndex: true, Downsample: downsample, Cutoff: cutoff, CutoffKV: cutoffKV,
		}, lazy)
		if err != nil {
			return nil, err
		}
		edsMetadata := bcf.Header.SpectraData[idx]

		var shape [3]int
		var dataForItem interface{}
		if lazy {
			dataForItem = lazyCube
			// We still need the shape for axes; re-derive it the same way
			// ParseHypermap does, without decoding.
			shape = hypermapShape(bcf.Header, edsMetadata, idx, downsample, cutoff, cutoffKV)
		} else {
			dataForItem = cube
			shape = cube.Shape
		}

		elemNames := make([]string, 0, len(bcf.Header.Elements))
		for name := range bcf.Header.Elements {
			elemNames = append(elemNames, name)
		}
		sort.Strings(elemNames)
		xrayLines := genElemList(bcf.Header.Elements)
		sort.Strings(xrayLines)

		item := &HspyItem{
			Data: dataForItem,
			Axes: []Axis{
				{Name: "height", Size: shape[0], Offset: 0, Scale: bcf.Header.YRes * float64(downsample), Units: bcf.Header.Units},
				{Name: "width", Size: shape[1], Offset: 0, Scale: bcf.Header.YRes * float64(downsample), Units: bcf.Header.Units},
				{Name: "Energy", Size: shape[2], Offset: edsMetadata.Offset, Scale: edsMetadata.Scale, Units: "keV"},
			},
			Metadata: map[string]interface{}{
				"Acquisition_instrument": map[string]interface{}{
					mode: bcf.Header.GetAcqInstrumentDict(true, idx),
				},
				"General": map[string]interface{}{
					"original_filename": filepath.Base(bcf.Filename),
					"title":             "EDX",
					"date":              bcf.Header.Date,
					"time":              bcf.Header.Time,
				},
				"Sample": map[string]interface{}{
					"name":       bcf.Header.Name,
					"elements":   elemNames,
					"xray_lines": xrayLines,
				},
				"Signal": map[string]interface{}{
					"signal_type": "EDS_" + mode,
					"record_by":   "spectrum",
					"quantity":    "X-rays (Counts)",
				},
			},
			OriginalMetadata: map[string]interface{}{
				"Hardware":          edsMetadata.HardwareMetadata,
				"Detector":          edsMetadata.DetectorMetadata,
				"Analysis":          edsMetadata.EsmaMetadata,
				"Spectrum":          edsMetadata.SpectrumMetadata,
				"DSP Configuration": bcf.Header.DspMetadata,
				"Line counter":      bcf.Header.LineCounter,
				"Stage":             bcf.Header.StageMetadata,
				"Microscope":        bcf.Header.SemMetadata,
				"mapping":           mapping,
			},
		}
		out = append(out, item)
	}
	return out, nil
}

// hypermapShape re-derives the (height, width, channels) shape without
// decoding pixel data, used only for populating axis metadata in the lazy
// path (where ParseHypermap does not eagerly compute a *HyperCube).
func hypermapShape(h *HyperHeader, eds *EDXSpectrum, index, downsample int, cutoff CutoffMode, cutoffKV float64) [3]int {
	var maxChan int
	switch cutoff {
	case CutoffFixedKV:
		maxChan = eds.EnergyToChannel(cutoffKV, true)
	case CutoffAuto:
		maxChan = eds.LastNonZeroChannel()
	case CutoffLowestConstraint:
		maxChan = h.EstimateMapChannels(index)
	default:
		maxChan = len(eds.Data) + 1
	}
	return [3]int{
		ceilDiv(h.Image.Height, downsample),
		ceilDiv(h.Image.Width, downsample),
		maxChan,
	}
}

func genElemList(elements map[string]ElementInfo) []string {
	out := make([]string, 0, len(elements))
	for name, info := range elements {
		out = append(out, name+"_"+parseLine(info.Line))
	}
	return out
}

func parseLine(lineString string) string {
	switch {
	case len(lineString) == 0:
		return ""
	case len(lineString) == 1:
		lineString += "a"
	case len(lineString) > 2:
		lineString = lineString[:2]
	}
	return strings.ToUpper(lineString[:1]) + strings.ToLower(lineString[1:])
}

func getMapping(mode string) map[string][2]interface{} {
	return map[string][2]interface{}{
		"Stage.Rotation": {fmt.Sprintf("Acquisition_instrument.%s.Stage.rotation", mode), nil},
		"Stage.Tilt":     {fmt.Sprintf("Acquisition_instrument.%s.Stage.tilt_alpha", mode), nil},
		"Stage.X":        {fmt.Sprintf("Acquisition_instrument.%s.Stage.x", mode), nil},
		"Stage.Y":        {fmt.Sprintf("Acquisition_instrument.%s.Stage.y", mode), nil},
		"Stage.Z":        {fmt.Sprintf("Acquisition_instrument.%s.Stage.z", mode), nil},
	}
}
