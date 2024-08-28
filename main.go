package main

import (
	ulua "github.com/thesoulless/watchmyback/internal/lua"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

func init() {
	ulua.L = lua.NewState()
	ulua.L.SetGlobal("import", luar.New(ulua.L, LuaImport))
	ulua.L.SetGlobal("require", luar.New(ulua.L, LuaImport))
}

func main() {

	// l := lua.NewState()
	// lua.OpenLibraries(l)
	// if err := lua.DoFile(l, "hello.lua"); err != nil {
	// 	panic(err)
	// }

	defer ulua.L.Close()
	if err := ulua.L.DoFile("hello.lua"); err != nil {
		panic(err)
	}

}

func LuaImport(pkg string) *lua.LTable {
	return ulua.Import(pkg)
}
