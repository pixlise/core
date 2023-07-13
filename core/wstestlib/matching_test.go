package wstestlib

import (
	"fmt"
	"time"
)

func Example_getDefinitionBetween() {
	fmt.Println(getDefinitionBetween("${USERID}", "${", "}"))
	fmt.Println(getDefinitionBetween("pre${USERID}", "${", "}"))
	fmt.Println(getDefinitionBetween("${USERID}post", "${", "}"))
	fmt.Println(getDefinitionBetween("${IDCHK=someValue}", "${", "}"))
	fmt.Println(getDefinitionBetween("${IDCHK=someValue,MODE=32,More=Less}", "${", "}"))
	fmt.Println(getDefinitionBetween("${IDCHK=someValue, MODE = 32,More=Less}", "${", "}"))
	fmt.Println(getDefinitionBetween("pre${IDCHK=someValue, MODE = 32,More=Less}post", "${", "}"))
	fmt.Println(getDefinitionBetween("${hello-", "${", "}"))
	fmt.Println(getDefinitionBetween("${}", "${", "}"))
	fmt.Println(getDefinitionBetween("pre${}", "${", "}"))
	fmt.Println(getDefinitionBetween("${}post", "${", "}"))
	fmt.Println(getDefinitionBetween("pre${}post", "${", "}"))
	fmt.Println(getDefinitionBetween("two${}defs${}here", "${", "}"))
	fmt.Println(getDefinitionBetween("no tokens!", "${", "}"))

	// Output:
	// USERID   <nil>
	// USERID pre  <nil>
	// USERID  post <nil>
	// IDCHK=someValue   <nil>
	// IDCHK=someValue,MODE=32,More=Less   <nil>
	// IDCHK=someValue, MODE = 32,More=Less   <nil>
	// IDCHK=someValue, MODE = 32,More=Less pre post <nil>
	//    failed to find closing token for "}" in "${hello-"
	//    <nil>
	//  pre  <nil>
	//   post <nil>
	//  pre post <nil>
	//  two defs${}here <nil>
	//  no tokens!  <nil>
}

func Example_parseDefinitions() {
	fmt.Println(parseDefinitions("${USERID}"))
	fmt.Println(parseDefinitions("${IDCHK=someValue}"))
	fmt.Println(parseDefinitions("${IDCHK=someValue,MODE=32,More=Less}"))
	fmt.Println(parseDefinitions("pre${IDCHK=someValue, MODE = 32,More=Less}post"))
	fmt.Println(parseDefinitions("${hello-"))
	fmt.Println(parseDefinitions("${}"))
	fmt.Println(parseDefinitions("no tokens!"))

	// Output:
	// map[USERID:]   <nil>
	// map[IDCHK:someValue]   <nil>
	// map[IDCHK:someValue MODE:32 More:Less]   <nil>
	// map[IDCHK:someValue MODE:32 More:Less] pre post <nil>
	// map[]   failed to find closing token for "}" in "${hello-"
	// map[]   <nil>
	// map[] no tokens!  <nil>
}

func Example_compareExpectedString() {
	ctx := compareParams{userId: "user123", savedItems: map[string]string{"name": "value", "name2": "value2"}, allowSaveItemOverwrite: false}

	fmt.Println(compareExpectedString("hello", "hello", ctx))
	fmt.Println(compareExpectedString("hello${}", "hello", ctx))
	fmt.Println(compareExpectedString("hello${USERID}", "hello", ctx))
	fmt.Println(compareExpectedString("${USERID}post", "hello", ctx))
	fmt.Println(compareExpectedString("${Chazwozza=2}", "hello", ctx))
	fmt.Println(compareExpectedString("${USERID}", "hello", ctx))
	fmt.Println(compareExpectedString("${USERID=7}", "hello", ctx))
	fmt.Println(compareExpectedString("${IGNORE}", "hello", ctx))
	fmt.Println(compareExpectedString("${IGNORE=9}", "hello", ctx))
	fmt.Println(compareExpectedString("${IDCHK=unknown}", "hello", ctx))
	fmt.Println(compareExpectedString("${IDCHK=name2}", "hello", ctx))
	fmt.Println(compareExpectedString("${IDSAVE=saveme}", "id119922", ctx))
	fmt.Println(ctx.savedItems)
	// Don't allow overwrite
	fmt.Println(compareExpectedString("${IDSAVE=saveme}", "998877", ctx))
	fmt.Println(compareExpectedString("${IDCHK=saveme}", "hello", ctx))
	ctx.allowSaveItemOverwrite = true
	fmt.Println(compareExpectedString("${IDSAVE=saveme}", "10000", ctx))
	fmt.Println(compareExpectedString("${IDCHK=saveme}", "hello", ctx))
	fmt.Println(compareExpectedString("${IDSAVE=shouldfail}", "", ctx))
	fmt.Println(compareExpectedString("${REGEXMATCH=hel+o}", "helllllo", ctx))
	fmt.Println(compareExpectedString("${REGEXMATCH=hel+o}", "hell_llo", ctx))
	fmt.Println(compareExpectedString("${SECAGO=7}", "hello", ctx))
	fmt.Println(compareExpectedString("${SECAGO=-7}", "1234", ctx))
	fmt.Println(compareExpectedString("${SECAGO=Yellow}", "1234", ctx))
	nowUnix := time.Now().Unix()
	recv, err := compareExpectedString("${SECAGO=7}", "123", ctx)
	fmt.Printf("%v, %v\n", recv, err.Error() == fmt.Sprintf("received time stamp 123 is %v seconds too old", nowUnix-123-7))
	nowStr := fmt.Sprintf("%v", nowUnix)
	recv, err = compareExpectedString("${SECAGO=7}", nowStr, ctx)
	fmt.Printf("%v, %v\n", nowStr == recv, err)

	fmt.Println(compareExpectedString("${SECAFTER=123}", "hello", ctx))
	fmt.Println(compareExpectedString("${SECAFTER=green}", "123", ctx))
	fmt.Println(compareExpectedString("${SECAFTER=123}", "122", ctx))
	fmt.Println(compareExpectedString("${SECAFTER=123}", "123", ctx))
	fmt.Println(compareExpectedString("${SECAFTER=123}", "124", ctx))

	// Output:
	// hello <nil>
	// hello <nil>
	//  Unexpected text around definition: hello${USERID}
	//  Unexpected text around definition: ${USERID}post
	//  Unknown matching cmd/param combination: ${Chazwozza=2}
	// user123 <nil>
	//  Unknown matching cmd/param combination: ${USERID=7}
	// hello <nil>
	//  Unknown matching cmd/param combination: ${IGNORE=9}
	//  failed to find defined id name to compare: ${IDCHK=unknown}
	// value2 <nil>
	// id119922 <nil>
	// map[name:value name2:value2 saveme:id119922]
	// saved id for saveme already exists: id119922, doesn't match save attempt: 998877
	//  saved id for saveme already exists: id119922, doesn't match save attempt: 998877
	// id119922 <nil>
	// 10000 <nil>
	// 10000 <nil>
	//  received empty string when trying to save id as "shouldfail"
	// helllllo <nil>
	//  received "hell_llo" did not match regex "hel+o"
	//  failed to parse received "hello" for "SECAGO" comparison as int: strconv.Atoi: parsing "hello": invalid syntax
	//  invalid value for SECAGO: "${SECAGO=-7}"
	//  failed to parse param "Yellow" for "SECAGO" as int: strconv.Atoi: parsing "Yellow": invalid syntax
	// , true
	// true, <nil>
	//  failed to parse received "hello" for "SECAFTER" comparison as int: strconv.Atoi: parsing "hello": invalid syntax
	//  failed to parse param "green" for "SECAFTER" as int: strconv.Atoi: parsing "green": invalid syntax
	//  received time stamp 122 is before expected 123
	// 123 <nil>
	// 124 <nil>
}

func Example_compareString() {
	ctx := compareParams{userId: "user123", savedItems: map[string]string{"name": "value", "name2": "value2"}, allowSaveItemOverwrite: false}

	// Some simple values
	fmt.Println(compare("hello", "world", ctx))
	fmt.Println(compare("hello", 1.7, ctx))
	fmt.Println(compare("hello", "hello", ctx))

	fmt.Println(compare("hello", "${IGNORE", ctx))
	fmt.Println(compare("hello", "${IGNORE}", ctx))
	fmt.Println(compare("hello", "${USERID}", ctx))
	fmt.Println(compare("user123", "${USERID}", ctx))

	// Output:
	// expected "world", received "hello"
	// expected "1.7", received "hello"
	// <nil>
	// failed to find closing token for "}" in "${IGNORE"
	// <nil>
	// expected "user123" (raw string: "${USERID}"), received "hello"
	// <nil>
}

func Example_compareFloat() {
	ctx := compareParams{userId: "user123", savedItems: map[string]string{"name": "value", "name2": "value2"}, allowSaveItemOverwrite: false}

	fmt.Println(compare(1.2, 1.7, ctx))
	fmt.Println(compare(1.2, nil, ctx))
	fmt.Println(compare(1.2, 1.2, ctx))

	// Output:
	// expected "1.7", received "1.2"
	// expected "<nil>", received "1.2"
	// <nil>
}

func Example_compareNil() {
	ctx := compareParams{userId: "user123", savedItems: map[string]string{"name": "value", "name2": "value2"}, allowSaveItemOverwrite: false}

	fmt.Println(compare(nil, "hello", ctx))
	fmt.Println(compare(174, nil, ctx))
	fmt.Println(compare(nil, nil, ctx))

	// Output:
	// expected "hello", received "<nil>"
	// expected "<nil>", received "174"
	// <nil>
}

func Example_compareBool() {
	ctx := compareParams{userId: "user123", savedItems: map[string]string{"name": "value", "name2": "value2"}, allowSaveItemOverwrite: false}

	fmt.Println(compare(true, "hello", ctx))
	fmt.Println(compare(174, true, ctx))
	fmt.Println(compare(true, false, ctx))
	fmt.Println(compare(true, true, ctx))

	// Output:
	// expected "hello", received "true"
	// expected "true", received "174"
	// expected "false", received "true"
	// <nil>
}

func Example_compareListSimple() {
	ctx := compareParams{userId: "user123", savedItems: map[string]string{"name": "value", "name2": "value2"}, allowSaveItemOverwrite: false}

	fmt.Println(compare([]string{"hello", "world"}, nil, ctx))
	// NOTE: doesn't work with strings, needs to be interface...
	fmt.Println(compare(72, []string{"hello", "world"}, ctx))
	fmt.Println(compare(72, []interface{}{"hello", "world"}, ctx))
	fmt.Println(compare([]string{"hello", "world"}, []interface{}{"hello", "world"}, ctx))
	fmt.Println(compare([]interface{}{"hello", "world"}, []interface{}{"hello", "world"}, ctx))
	fmt.Println(compare([]interface{}{"world", "hello"}, []interface{}{"hello", "world"}, ctx))

	// Output:
	// expected "<nil>", received "[hello world]"
	// unexpected type: []string for defined expected data
	// expected "[hello world]", received "72"
	// expected "[hello world]", received "[hello world]"
	// <nil>
	// "hello": expected "hello", received "world"
}

func Example_compareMaps() {
	ctx := compareParams{userId: "user123", savedItems: map[string]string{"name": "value", "name2": "value2"}, allowSaveItemOverwrite: false}

	fmt.Println(compare(72, map[string]interface{}{"hello": "world", "zztop": "band"}, ctx))
	fmt.Println(compare(map[string]interface{}{"hello": "world", "zztop": "band"}, map[string]interface{}{"hello": "world", "zztop": "band"}, ctx))
	fmt.Println(compare(map[string]interface{}{"hello": "world", "acdc": "band"}, map[string]interface{}{"hello": "world", "zztop": "band"}, ctx))
	fmt.Println(compare(map[string]interface{}{"zztop": "band", "hello": "world"}, map[string]interface{}{"hello": "${IGNORE}", "zztop": "band"}, ctx))
	fmt.Println(compare(map[string]interface{}{"zztop": 17, "hello": "world"}, map[string]interface{}{"hello": "world", "zztop": "${IGNORE}"}, ctx))
	fmt.Println(compare(map[string]interface{}{"zztop": []bool{true, false}, "hello": "world"}, map[string]interface{}{"hello": "world", "zztop": "${IGNORE}"}, ctx))

	// Output:
	// expected "map[hello:world zztop:band]", received "72"
	// <nil>
	// expected key: "hello", received key: "acdc"
	// <nil>
	// <nil>
	// <nil>
}

func Example_compareMapsOfLists() {
	ctx := compareParams{userId: "user123", savedItems: map[string]string{"name": "value", "name2": "value2"}, allowSaveItemOverwrite: false}

	// List comparison
	// NOTE: testing here with doubles, ints fail, but JSON decode should never return an int anyway
	fmt.Println(compare(map[string]interface{}{"zztop": 17.2, "hello": []int{1, 2, 3}}, map[string]interface{}{"helloo${LIST,MINLENGTH=4}": []int{}, "zztop": 17.2}, ctx))
	// Expected map value not a list
	fmt.Println(compare(map[string]interface{}{"zztop": 17.2, "hello": []int{1, 2, 3}}, map[string]interface{}{"hello${LIST,MINLENGTH=4}": "shouldbelist", "zztop": 17.2}, ctx))
	// Expected value is an int, which we don't support due to json parser only returning doubles...
	fmt.Println(compare(map[string]interface{}{"zztop": 17.2, "hello": []int{1, 2, 3}}, map[string]interface{}{"hello${LIST,MINLENGTH=4}": []int{}, "zztop": 17.2}, ctx))
	// Received type is not a list
	fmt.Println(compare(map[string]interface{}{"zztop": 17.2, "hello": "should-be-list"}, map[string]interface{}{"hello${LIST,MINLENGTH=4}": []interface{}{}, "zztop": 17.2}, ctx))
	// Not enough items
	fmt.Println(compare(map[string]interface{}{"zztop": 17.2, "hello": []interface{}{1, 2, 3}}, map[string]interface{}{"hello${LIST,MINLENGTH=4}": []interface{}{}, "zztop": 17.2}, ctx))
	// Missing mode
	fmt.Println(compare(map[string]interface{}{"zztop": 17.2, "hello": []interface{}{1, 2, 3, 4}}, map[string]interface{}{"hello${LIST,MINLENGTH=4}": []interface{}{}, "zztop": 17.2}, ctx))

	// Check with ints, should fail on a type error
	fmt.Println(compare(map[string]interface{}{"zztop": 17, "hello": []interface{}{1, 2, 3, 4}}, map[string]interface{}{"hello${LIST,MODE=LENGTH,MINLENGTH=4}": []interface{}{}, "zztop": 17}, ctx))

	// Back to doubles, this should work
	fmt.Println(compare(map[string]interface{}{"zztop": 17.2, "hello": []interface{}{1, 2, 3, 4}}, map[string]interface{}{"hello${LIST,MODE=LENGTH,MINLENGTH=4}": []interface{}{}, "zztop": 17.2}, ctx))

	// Will fail because list contents are ints, but just says 1 not found, because it's actually getting an error from every comparison and can't
	// tell the difference...
	fmt.Println(compare(map[string]interface{}{"zztop": 17.2, "hello": []interface{}{1, 2, 3, 4}}, map[string]interface{}{"hello${LIST,MODE=CONTAINS,MINLENGTH=4}": []interface{}{1, 3}, "zztop": 17.2}, ctx))

	// With doubles in the list, it should work
	fmt.Println(compare(map[string]interface{}{"zztop": 17.2, "hello": []interface{}{1.2, 2.2, 3.2, 4.2}}, map[string]interface{}{"hello${LIST,MODE=CONTAINS,MINLENGTH=4}": []interface{}{1.2, 3.2}, "zztop": 17.2}, ctx))

	// Genuine item not found
	fmt.Println(compare(map[string]interface{}{"zztop": 17.2, "hello": []interface{}{1.2, 2.2, 3.2, 4.2}}, map[string]interface{}{"hello${LIST,MODE=CONTAINS,MINLENGTH=4}": []interface{}{1.2, 7.2}, "zztop": 17.2}, ctx))

	// Order shouldn't matter for list comparison
	fmt.Println(compare(map[string]interface{}{"zztop": 17.2, "hello": []interface{}{2.2, 4.2, 1.2, 3.2}}, map[string]interface{}{"hello${LIST,MODE=CONTAINS,MINLENGTH=4}": []interface{}{1.2, 3.2}, "zztop": 17.2}, ctx))
	// Checking with LENGTH not MINLENGTH
	fmt.Println(compare(map[string]interface{}{"zztop": 17.2, "hello": []interface{}{2.2, 4.2, 1.2, 3.2}}, map[string]interface{}{"hello${LIST,MODE=CONTAINS,LENGTH=4}": []interface{}{1.2, 3.2}, "zztop": 17.2}, ctx))
	fmt.Println(compare(map[string]interface{}{"zztop": 17.2, "hello": []interface{}{2.2, 4.2, 1.2, 3.2}}, map[string]interface{}{"hello${LIST,MODE=CONTAINS,LENGTH=3}": []interface{}{1.2, 3.2}, "zztop": 17.2}, ctx))
	fmt.Println(compare(map[string]interface{}{"zztop": 17.2, "hello": []interface{}{2.2, 4.2, 1.2, 3.2}}, map[string]interface{}{"hello${LIST,MODE=LENGTH,LENGTH=4}": []interface{}{}, "zztop": 17.2}, ctx))
	fmt.Println(compare(map[string]interface{}{"zztop": 17.2, "hello": []interface{}{2.2, 4.2, 1.2, 3.2}}, map[string]interface{}{"hello${LIST,MODE=LENGTH}": []interface{}{3.3}, "zztop": 17.2}, ctx))

	// Checking with unrecognised list specification
	fmt.Println(compare(map[string]interface{}{"zztop": 17.2, "hello": []interface{}{2.2, 4.2, 1.2, 3.2}}, map[string]interface{}{"hello${LIST,MODE=CONTAINS,NICENESS=4}": []interface{}{1.2, 3.2}, "zztop": 17.2}, ctx))

	// Output:
	// expected key: "helloo", received key: "hello"
	// "hello${LIST,MINLENGTH=4}": expected list for list parse spec "map[MINLENGTH:4]"
	// "hello${LIST,MINLENGTH=4}": expected list for list parse spec "map[MINLENGTH:4]"
	// "hello${LIST,MINLENGTH=4}": expected list compatible with parse spec "map[MINLENGTH:4]", received "should-be-list"
	// "hello${LIST,MINLENGTH=4}": expected at least 4 list items, received 3
	// "hello${LIST,MINLENGTH=4}": invalid mode in list compare specifications: map[MINLENGTH:4]
	// "zztop": unexpected type: int for defined expected data
	// <nil>
	// "hello${LIST,MODE=CONTAINS,MINLENGTH=4}": expected list to contain item "1"
	// <nil>
	// "hello${LIST,MODE=CONTAINS,MINLENGTH=4}": expected list to contain item "7.2"
	// <nil>
	// <nil>
	// "hello${LIST,MODE=CONTAINS,LENGTH=3}": expected exactly 3 list items, received 4
	// <nil>
	// "hello${LIST,MODE=LENGTH}": expected 1 list items, received 4
	// "hello${LIST,MODE=CONTAINS,NICENESS=4}": unrecognised list spec: NICENESS
}

func Example_compareMapWithKeyDef() {
	ctx := compareParams{userId: "user123", savedItems: map[string]string{"name": "value", "name2": "value2"}, allowSaveItemOverwrite: false}

	fmt.Println(compare(map[string]interface{}{"zztop": []bool{true, false}, "user123": "world"}, map[string]interface{}{"${USERID}": "world", "zztop": "${IGNORE}"}, ctx))

	// Output:
	// <nil>
}
