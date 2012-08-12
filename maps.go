/*
Copyright (c) 2011, 2012 Andrew Wilkins <axwalk@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
of the Software, and to permit persons to whom the Software is furnished to do
so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package llgo

import (
	"github.com/axw/gollvm/llvm"
	"github.com/axw/llgo/types"
)

// mapLookup searches a map for a specified key, returning a pointer to the
// memory location for the value. If insert is given as true, and the key
// does not exist in the map, it will be added with an uninitialised value.
func (c *compiler) mapLookup(m *LLVMValue, key Value, insert bool) *LLVMValue {
	mapType := m.Type().(*types.Map)
	maplookup := c.module.Module.NamedFunction("runtime.maplookup")
	ptrType := c.target.IntPtrType()
	if maplookup.IsNil() {
		// params: dynamic type, mapptr, keyptr, insertifmissing
		paramTypes := []llvm.Type{ptrType, ptrType, ptrType, llvm.Int1Type()}
		funcType := llvm.FunctionType(ptrType, paramTypes, false)
		maplookup = llvm.AddFunction(c.module.Module, "runtime.maplookup", funcType)
	}
	args := make([]llvm.Value, 4)
	args[0] = llvm.ConstPtrToInt(c.types.ToRuntime(m.Type()), ptrType)
	args[1] = c.builder.CreatePtrToInt(m.pointer.LLVMValue(), ptrType, "")
	if insert {
		args[3] = llvm.ConstAllOnes(llvm.Int1Type())
	} else {
		args[3] = llvm.ConstNull(llvm.Int1Type())
	}

	if lv, islv := key.(*LLVMValue); islv && lv.pointer != nil {
		args[2] = c.builder.CreatePtrToInt(lv.pointer.LLVMValue(), ptrType, "")
	}
	if args[2].IsNil() {
		stackval := c.builder.CreateAlloca(c.types.ToLLVM(key.Type()), "")
		c.builder.CreateStore(key.LLVMValue(), stackval)
		args[2] = c.builder.CreatePtrToInt(stackval, ptrType, "")
	}

	eltPtrType := &types.Pointer{Base: mapType.Elt}
	llvmtyp := c.types.ToLLVM(eltPtrType)
	zeroglobal := llvm.AddGlobal(c.module.Module, llvmtyp.ElementType(), "")
	zeroglobal.SetInitializer(llvm.ConstNull(llvmtyp.ElementType()))
	result := c.builder.CreateCall(maplookup, args, "")
	result = c.builder.CreateIntToPtr(result, llvmtyp, "")
	notnull := c.builder.CreateIsNotNull(result, "")
	result = c.builder.CreateSelect(notnull, result, zeroglobal, "")
	value := c.NewLLVMValue(result, eltPtrType)
	return value.makePointee()
}

func (c *compiler) mapDelete(m *LLVMValue, key Value) {
	mapdelete := c.module.Module.NamedFunction("runtime.mapdelete")
	ptrType := c.target.IntPtrType()
	if mapdelete.IsNil() {
		// params: dynamic type, mapptr, keyptr
		paramTypes := []llvm.Type{ptrType, ptrType, ptrType}
		funcType := llvm.FunctionType(llvm.VoidType(), paramTypes, false)
		mapdelete = llvm.AddFunction(c.module.Module, "runtime.mapdelete", funcType)
	}
	args := make([]llvm.Value, 3)
	args[0] = llvm.ConstPtrToInt(c.types.ToRuntime(m.Type()), ptrType)
	args[1] = c.builder.CreatePtrToInt(m.pointer.LLVMValue(), ptrType, "")
	if lv, islv := key.(*LLVMValue); islv && lv.pointer != nil {
		args[2] = c.builder.CreatePtrToInt(lv.pointer.LLVMValue(), ptrType, "")
	}
	if args[2].IsNil() {
		stackval := c.builder.CreateAlloca(c.types.ToLLVM(key.Type()), "")
		c.builder.CreateStore(key.LLVMValue(), stackval)
		args[2] = c.builder.CreatePtrToInt(stackval, ptrType, "")
	}
	c.builder.CreateCall(mapdelete, args, "")
}

// mapNext iterates through a map, accepting an iterator state value,
// and returning a new state value, key pointer, and value pointer.
func (c *compiler) mapNext(m *LLVMValue, nextin llvm.Value) (nextout, pk, pv llvm.Value) {
	mapnext := c.module.Module.NamedFunction("runtime.mapnext")
	ptrType := c.target.IntPtrType()
	if mapnext.IsNil() {
		// params: dynamic type, mapptr, nextptr
		paramTypes := []llvm.Type{ptrType, ptrType, ptrType}
		// results: nextptr, keyptr, valptr
		resultType := llvm.StructType([]llvm.Type{ptrType, ptrType, ptrType}, false)
		funcType := llvm.FunctionType(resultType, paramTypes, false)
		mapnext = llvm.AddFunction(c.module.Module, "runtime.mapnext", funcType)
	}
	args := make([]llvm.Value, 3)
	args[0] = llvm.ConstPtrToInt(c.types.ToRuntime(m.Type()), ptrType)
	args[1] = c.builder.CreatePtrToInt(m.pointer.LLVMValue(), ptrType, "")
	args[2] = nextin
	results := c.builder.CreateCall(mapnext, args, "")
	nextout = c.builder.CreateExtractValue(results, 0, "")
	pk = c.builder.CreateExtractValue(results, 1, "")
	pv = c.builder.CreateExtractValue(results, 2, "")

	keyptrtype := &types.Pointer{Base: m.Type().(*types.Map).Key.(types.Type)}
	valptrtype := &types.Pointer{Base: m.Type().(*types.Map).Elt.(types.Type)}
	pk = c.builder.CreateIntToPtr(pk, c.types.ToLLVM(keyptrtype), "")
	pv = c.builder.CreateIntToPtr(pv, c.types.ToLLVM(valptrtype), "")

	return
}
