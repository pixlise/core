package brukerreader

import (
	"bytes"
	"errors"
	"io"
	"strings"
)

// ---------------------------------------------------------------------
// XMLNode: a generic, order-preserving XML tree used to reproduce
// ElementTree-style traversal (t.find("./Tag[@Attr='Val']"), t.attrib,
// t.text, list(t)) without hard-coding a schema.
// ---------------------------------------------------------------------

type XMLAttr struct {
	Name  string
	Value string
}

type XMLNode struct {
	Tag      string
	Attrs    []XMLAttr
	Text     string
	Children []*XMLNode
}

func (n *XMLNode) Attr(name string) (string, bool) {
	for _, a := range n.Attrs {
		if a.Name == name {
			return a.Value, true
		}
	}
	return "", false
}

// Find returns the first direct child with the given tag. If attrName is
// non-empty, the child must also have that attribute equal to attrVal.
// This mirrors the small subset of ElementTree XPath used by this format:
// "./Tag" and "./Tag[@Attr='Val']".
func (n *XMLNode) Find(tag, attrName, attrVal string) *XMLNode {
	if n == nil {
		return nil
	}
	for _, c := range n.Children {
		if c.Tag != tag {
			continue
		}
		if attrName == "" {
			return c
		}
		if v, ok := c.Attr(attrName); ok && v == attrVal {
			return c
		}
	}
	return nil
}

// FindAll returns every direct child matching tag (and, optionally,
// attrName == attrVal).
func (n *XMLNode) FindAll(tag, attrName, attrVal string) []*XMLNode {
	var out []*XMLNode
	if n == nil {
		return out
	}
	for _, c := range n.Children {
		if c.Tag != tag {
			continue
		}
		if attrName == "" {
			out = append(out, c)
			continue
		}
		if v, ok := c.Attr(attrName); ok && v == attrVal {
			out = append(out, c)
		}
	}
	return out
}

// FindPath walks a sequence of (tag, attrName, attrVal) triples, i.e. a
// flattened version of the multi-level "./A/B[@Type='C']/D" XPaths used in
// the original Python source.
type PathStep struct {
	Tag, AttrName, AttrVal string
}

func (n *XMLNode) FindPath(steps ...PathStep) *XMLNode {
	cur := n
	for _, s := range steps {
		cur = cur.Find(s.Tag, s.AttrName, s.AttrVal)
		if cur == nil {
			return nil
		}
	}
	return cur
}

// parseXML is a minimal streaming XML parser producing an *XMLNode tree.
// It intentionally ignores namespaces/comments/PIs, which Bruker's metadata
// XML does not use.
func parseXML(data []byte) (*XMLNode, error) {
	dec := newXMLDecoder(data)
	var stack []*XMLNode
	var root *XMLNode
	for {
		tok, err := dec.next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch tok.kind {
		case tokStart:
			node := &XMLNode{Tag: tok.name}
			for _, a := range tok.attrs {
				node.Attrs = append(node.Attrs, XMLAttr{Name: a.name, Value: a.value})
			}
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, node)
			} else {
				root = node
			}
			if !tok.selfClose {
				stack = append(stack, node)
			}
		case tokEnd:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case tokText:
			if len(stack) > 0 {
				stack[len(stack)-1].Text += tok.text
			}
		}
	}
	if root == nil {
		return nil, errors.New("bruker: empty or invalid XML document")
	}
	return root, nil
}

// --- tiny hand-rolled XML tokenizer -----------------------------------
// encoding/xml's generic-tree support is awkward for this dynamic schema,
// and Bruker's XML is simple (no namespaces, no CDATA tricks beyond
// standard entities), so a small dedicated tokenizer keeps this dependency
// free and predictable.

type tokKind int

const (
	tokStart tokKind = iota
	tokEnd
	tokText
)

type xmlAttrTok struct{ name, value string }

type xmlTok struct {
	kind      tokKind
	name      string
	attrs     []xmlAttrTok
	text      string
	selfClose bool
}

type xmlDecoder struct {
	data []byte
	pos  int
}

func newXMLDecoder(data []byte) *xmlDecoder { return &xmlDecoder{data: data} }

func (d *xmlDecoder) next() (xmlTok, error) {
	if d.pos >= len(d.data) {
		return xmlTok{}, io.EOF
	}
	if d.data[d.pos] == '<' {
		// skip declarations/comments/PIs
		for {
			if d.pos >= len(d.data) {
				return xmlTok{}, io.EOF
			}
			if bytes.HasPrefix(d.data[d.pos:], []byte("<?")) {
				end := bytes.Index(d.data[d.pos:], []byte("?>"))
				if end < 0 {
					return xmlTok{}, errors.New("bruker: unterminated XML PI")
				}
				d.pos += end + 2
				continue
			}
			if bytes.HasPrefix(d.data[d.pos:], []byte("<!--")) {
				end := bytes.Index(d.data[d.pos:], []byte("-->"))
				if end < 0 {
					return xmlTok{}, errors.New("bruker: unterminated XML comment")
				}
				d.pos += end + 3
				continue
			}
			if bytes.HasPrefix(d.data[d.pos:], []byte("<!")) {
				end := bytes.IndexByte(d.data[d.pos:], '>')
				if end < 0 {
					return xmlTok{}, errors.New("bruker: unterminated XML doctype")
				}
				d.pos += end + 1
				continue
			}
			break
		}
		if d.pos >= len(d.data) {
			return xmlTok{}, io.EOF
		}
		if d.data[d.pos+1] == '/' {
			end := bytes.IndexByte(d.data[d.pos:], '>')
			if end < 0 {
				return xmlTok{}, errors.New("bruker: unterminated end tag")
			}
			d.pos += end + 1
			return xmlTok{kind: tokEnd}, nil
		}
		end := bytes.IndexByte(d.data[d.pos:], '>')
		if end < 0 {
			return xmlTok{}, errors.New("bruker: unterminated start tag")
		}
		raw := d.data[d.pos+1 : d.pos+end]
		d.pos += end + 1
		self := false
		if len(raw) > 0 && raw[len(raw)-1] == '/' {
			self = true
			raw = raw[:len(raw)-1]
		}
		return parseStartTag(raw, self)
	}
	end := bytes.IndexByte(d.data[d.pos:], '<')
	if end < 0 {
		end = len(d.data) - d.pos
	}
	text := unescapeXML(string(d.data[d.pos : d.pos+end]))
	d.pos += end
	return xmlTok{kind: tokText, text: text}, nil
}

func parseStartTag(raw []byte, selfClose bool) (xmlTok, error) {
	s := string(bytes.TrimSpace(raw))
	i := 0
	for i < len(s) && !isSpace(s[i]) {
		i++
	}
	name := s[:i]
	tok := xmlTok{kind: tokStart, name: name, selfClose: selfClose}
	rest := strings.TrimSpace(s[i:])
	for len(rest) > 0 {
		eq := strings.IndexByte(rest, '=')
		if eq < 0 {
			break
		}
		attrName := strings.TrimSpace(rest[:eq])
		rest = strings.TrimLeft(rest[eq+1:], " \t\r\n")
		if len(rest) == 0 {
			break
		}
		quote := rest[0]
		if quote != '\'' && quote != '"' {
			break
		}
		endQ := strings.IndexByte(rest[1:], quote)
		if endQ < 0 {
			break
		}
		val := unescapeXML(rest[1 : 1+endQ])
		tok.attrs = append(tok.attrs, xmlAttrTok{name: attrName, value: val})
		rest = strings.TrimSpace(rest[1+endQ+1:])
	}
	return tok, nil
}

func isSpace(b byte) bool { return b == ' ' || b == '\t' || b == '\r' || b == '\n' }

func unescapeXML(s string) string {
	if !strings.ContainsRune(s, '&') {
		return s
	}
	replacer := strings.NewReplacer(
		"&lt;", "<", "&gt;", ">", "&quot;", `"`, "&apos;", "'", "&amp;", "&",
	)
	return replacer.Replace(s)
}
