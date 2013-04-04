// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package locale_test

import (
	"code.google.com/p/go.exp/locale"
	"fmt"
)

func ExampleID_Parent() {
	loc := locale.Make("sl-Latn-IT-nedis")
	fmt.Println(loc.Parent())
	// TODO:Output: sl-Latn-IT
}

func ExampleID_Written() {
	loc := locale.Make("sl-Latn-IT-nedis")
	fmt.Println(loc.Written())
	// TODO:Output: sl-Latn
}

func ExampleID_Script() {
	en := locale.Make("en")
	sr := locale.Make("sr")
	fmt.Println(en.Script())
	fmt.Println(sr.Script())
	// TODO:Output:
	// Latn High
	// Cyrl Low
}

func ExampleID_Part() {
	loc := locale.Make("sr-RS")
	script := loc.Part(locale.ScriptPart)
	region := loc.Part(locale.RegionPart)
	fmt.Printf("%q %q", script, region)
	// TODO:Output: "" "RS"
}

func ExampleID_Scope() {
	loc := locale.Make("sr")
	set := loc.Scope()
	fmt.Println(set.Locales())
	fmt.Println(set.Languages())
	fmt.Println(set.Scripts())
	fmt.Println(set.Regions())
	// TODO:Output:
	// [sr_Cyrl sr_Cyrl_ME sr_Latn sr_Latn_ME sr_Cyrl_BA sr_Cyrl_RS sr_Latn_BA sr_Latn_RS]
	// [sr]
	// [Cyrl Latn]
	// [BA ME RS]
}

func ExampleScript_Scope() {
	loc := locale.Make("zen-Tfng")
	script, _ := loc.Script()
	set := script.Scope()
	fmt.Println(set.Locales())
	fmt.Println(set.Languages())
	fmt.Println(set.Scripts())
	fmt.Println(set.Regions())
	// TODO:Output:
	// [shi shi-Tfng shi-Tfng_MA tzm]
	// [shi tzm zen]
	// [Tfng]
	// [MA]
}
