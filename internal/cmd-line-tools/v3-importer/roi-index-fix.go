package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/indexcompression"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/protobuf/proto"
)

func fixROIIndexes(
	datasetBucket string,
	fs fileaccess.FileAccess,
	dest *mongo.Database) error {
	coll := dest.Collection(dbCollections.RegionsOfInterestName)

	// Loop through all ROIs, form commands to edit them
	filter := bson.D{}
	opts := options.Find()
	cursor, err := coll.Find(context.TODO(), filter, opts)
	if err != nil {
		fatalError(err)
	}

	allRois := []*protos.ROIItem{}
	err = cursor.All(context.TODO(), &allRois)
	if err != nil {
		fatalError(err)
	}

	rois := []*protos.ROIItem{}

	sharedROIIds := []string{"115k2i6c3nbhpcxu", "84s0c2x665iygzyu", "73x1lr9nww8g794k", "kwhds2tza5r2t8bh", "3q2cj3xtcnt8jfzz", "ubuk9typoc1zn5ob", "hk2lky5drpjeh20f", "z5bwhw1iv3f3fxks", "e3sznvrrugam4yzh", "4toylk7lvh0zvpd4", "yqrwlpj4x3pj4ir1", "y20vl9fe7jkuppc8", "bhv7i4c18ax5wts8", "eobyavwd56az46r0", "yjn1cnj3kipwp8to", "7s9mq8ifuu60s18c", "eybdwjvuh4ei7rkd", "qgpifa2uj46aiuc1", "858v6966iwmufrwr", "3ovgf0m0k26xz5m5", "0klodzdcnnqp6mzw", "wnkbfwtolbo9i1so", "5qw9tul206vqk46g", "j33m0dj5okpo6wht", "5e6r85z45afv0a4y", "x61mj3av95amb2rw", "gb5c5n1kds4itf2r", "lc8jlatk8o6kfq5a", "6h9xflmdkb1bb9vb", "j0hf838vq83q1v00", "5a77juyyibq4u3ma", "5kfljfsfvvtknfql", "realhcgs4f8norwt", "4muy88180gknm5eg", "nsewwydset6qk3pk", "rsavhs25z8irminx", "ibpgnm0aymal545p", "ivmg0ipdndbeq7hx", "gwjh1d16mo10m1kk", "zrcm3xz2p29mfd8w", "tb99udf2dp7wji15", "0xe83e4mz0xumxuj", "92lmkjvbs9d1sxg9", "6c87zjc4so407dmf", "4p8i8m9o7kfjcxhd", "b06yvzr8xrdlhm28", "jh1l5mw1k1kq2baj", "6wyrtzyx0xtkw4z7", "qdviag6yrulekehi", "3l82meijuy2krgrt", "umrjaw5abxuf5nvs", "lxypotyqim5spl86", "281qiamw1dga78gz", "yrmbd6gk0ulhrfix", "xfviyk7cloencvde", "h5hu04tqo5m7mhcv", "5vsggyeo9zv53hcc", "m8tps5mykq423756", "1z5gpagn5mbx4a2l", "ljbswuzcsvre3t10", "4g52f7xxvrs8imet", "bekm45xdb3ndmjiy", "6wm8nwfcqc6j3wos", "x428f6kmd5cwtxin", "uijgsw0wu2hopayv", "dtma8ina0k8kxofc", "12q43vpd7s1uwxnn", "f0yv9wxd8sczt2ni", "tyrblmqb1flu1mc5", "jmt8jtv0pr8ljbge", "4k110xmuo3rfz7zi", "fs1l222f4voh8tfz", "uwhdcy84jmevmt0f", "nqr0o1ogdl8cp096", "84rpugh34nr9jkh9", "2bob4mkdqlfnt4jv", "e5884zndmwhel5zl", "76i4yk859ndoltn8", "tegbhs5g38p8g2z9", "8qm7rtwy0zwrtf0l", "g5sjaen08ctihqu7", "ipevzo7m9xo7kw84", "ux361v2z74c3fy31", "li2s0x4rj9h97syb", "fqie6yxvzl2sr2fe", "l9l6vviu85bq9rq0", "ufrpk8136t8xjum6", "9r0egpkmd8cc2lty", "80id7kjfoe89029v", "6rps714oamaraqrf", "w9pct5xunjngsaue", "7y34eps6o08mtm6f", "yl1sb0mjveqlkohl", "5mc9n9t7u95u2sj5", "zcuc4pg4m6b2ajow", "v449aujj9wmp6yyj", "63dsdr7lf4n5nqoz", "t2onoskmbk2le70w", "tmac2jvennro1p0a", "9nu627g182xln4cj", "0obmzxlnsi5udt3q", "e8fea0orfvi994s0", "hjmgbdgb4sivkgns", "v0a14teb52litgol", "j2dvb7t6s3o3kb44", "cj7oovap9ftcxljn", "rljvzwk1sy9c0tal", "doyeyzzczanplxhs", "abmmv0iy6c91bqc7", "prn21emvtbcsmoa7", "2dqgbk0bhl58x7kb", "j26k8pwp09741khg", "3x9111ei6ukl8w96", "9nj92b7x7jnm6hcz", "naeyfzmknlvsnp5i", "ibygzj7s66ljgvir", "xy9y542ryps8dnf8", "fb4g2ueufctwhm6p", "es4j151shewbxv00", "g642y7rxo3pywpeg", "vv0r6r3ifxfb3exy", "sguxueoc9ncl1bez", "14w7cxhcpnj0n0of", "6t0pbsd4x7mbjn0a", "gkldsprsul1sle9i", "vks3qbsbvy4qitu0", "8l1y4ewg77988uto", "i25ukobr9lxj6b87", "86gkocyyvesi0kwc", "6kvxlpyz3arh9tpo", "nrrhkckez0igjupx", "re9tkq5bli555ofs", "ikand6fzmc09c6ry", "m9qg4aizm0gp029q", "m3bbhot4ulqbd8pq", "vpr2dferuxi0bc46", "kupp7wwqdd09vgpg", "36yxai7c9h9se0xh", "mqamr41zhr24sqqr", "1qt6424qfwu0j6y0", "7wonc7a792xjbfcy", "rqzgc9l9f09zta2j", "3ubndh5wob0vuvts", "gfaulzv2qcvflnna", "rtg3z0een4xk8e64", "y3x3m23xprxhmdx8", "n7bg5d222aubxzvn", "ldene21z50xm9jr5", "zd2jwhzk5x6xugbw", "sj7cychc2wit901w", "4vjrtd1gn2j1abia", "r9xqm4741rddy1n2", "h869rrqv8h740n9n", "jcddjgo03aisomfj", "u17795t3hhe8zgw9", "xwieb60tsvyr6dbi", "e8v2toph0v0w23nf", "o8loq4xrghjiqdxc", "47a7xbrg8qbxa5a8", "l2jg7uhe1uen1nfv", "78qd1e1ol7ltp0t7", "zdcvrfdcrl7ox32v", "xdvz5gu2hbd9b493", "ojbnb0cgt3ruipbb", "xlondlhh41pnv3jg", "74gx9vnactyxd65x", "o29n8brjgo8xurj4", "roddd9fa2vnf6wnr", "m92gihaaf3mrwwpf", "3d8it28s26jt7s92", "qa438bkuktomsc2g", "9udqhmgk3fsjjh2y", "6upmbsy09ee2brkq", "0q0avwq34ky714i2", "z5ux9pnybn8j164a", "4enmpbqn8z821d8h", "61lsbr4s6ld97e3m", "iuuo1vwlc6gs7jrk", "2sbi6m0hl5i17bld", "2pzzywhyuvyvw6ts", "fwv7la4l0h01ayl7", "a65q4nvvk897ggc7", "908lqcajk510kspk", "hbbpsu1c0w9p0co8", "oylf95n3wztkfshi", "z216dj5i2sqeh5qf", "nm9a7fpz6u0wj9lw", "v544tpcpqasfj1h2", "wzajr022doem1x0a", "ysdi9pncv24j7cs9", "djnk6r3uwygk94qi", "hny2nw82fcczxl1z", "ir3lprpm16qduwsm", "wt2szlgmf6h3i3rt", "fydmbwksrs1bhipf", "xii7aedyb6vqe1q6", "sif8dmhypo1x2dgl", "1m76fdjilhoc43u6", "rbi54cv2iumnu8nc", "13ke8gza6htyn748", "qsnyrli55jda8wgv", "rrzar0li9nfozvke", "9qg6vgyttwh2fmvu", "b01ux07bf1bs0ibt", "7he6uggp5mcgpx4p", "u59zbhf6a92wz94f", "oom78qck2elc13xc", "f5tpzef3ncgiuiky", "mlxn151t8wrmc3m1", "hjtgkmnhgcrze3a0", "hsikjonta215tdal", "n8n35xghrabtxt5b", "kryh9ld5lpzwrkm1", "0hb8lmkmqzasr2i6", "ogput60ue9vv4xyi", "jie3wrf34n6lh0la", "4k5d5b4tlnecjauu"}
	sharedROIIdMap := map[string]bool{}
	for _, id := range sharedROIIds {
		sharedROIIdMap[id] = true
	}

	// Cache all DB files locally
	datasetIds := map[string]bool{}
	for _, roi := range allRois {
		if sharedROIIdMap[roi.Id] {
			datasetIds[roi.ScanId] = true
			rois = append(rois, roi)
		}
	}

	fmt.Printf("dataset IDs: %v\nrois: %v\n", len(datasetIds), len(rois))

	for scanId := range datasetIds {
		// Download the DB file
		ds, err := fs.ReadObject(datasetBucket, "Datasets/"+scanId+"/dataset.bin")
		if err != nil {
			fatalError(err)
		}

		exprPB := &protos.Experiment{}
		err = proto.Unmarshal(ds, exprPB)
		if err != nil {
			fatalError(err)
		}

		fmt.Printf("--> %v\n", scanId)

		for _, roi := range rois {
			if strings.Compare(roi.ScanId, scanId) == 0 {
				if !utils.ItemInSlice(roi.Id, sharedROIIds) {
					continue
				}

				// Read the stored location indexes, convert to PMCs
				pmcs := []int32{}

				locIdxs, err := indexcompression.DecodeIndexList(roi.ScanEntryIndexesEncoded, -1)
				if err != nil {
					fatalError(err)
				}

				for _, locIdx := range locIdxs {
					if locIdx >= 0 && locIdx < uint32(len(exprPB.Locations)) {
						loc := exprPB.Locations[locIdx]
						pmc, err := strconv.Atoi(loc.Id)
						if err != nil {
							fatalError(err)
						}
						pmcs = append(pmcs, int32(pmc))
					}
				}

				// Compress PMC list
				encodedPMCs, err := indexcompression.EncodeIndexList(pmcs)
				if err != nil {
					fatalError(err)
				}

				// Form a command to write the encoded PMCs
				if !utils.SlicesEqual(roi.ScanEntryIndexesEncoded, encodedPMCs) {
					nums := ""
					for _, num := range encodedPMCs {
						if len(nums) > 0 {
							nums += ","
						}
						nums += fmt.Sprintf("%v", num)
					}

					fmt.Printf("db.regionsOfInterest.update({_id:\"%v\"}, {$set: {\"scanentryindexesencoded\": [%v]}})\n", roi.Id, nums)
				}
			}
		}
	}

	return nil
}
