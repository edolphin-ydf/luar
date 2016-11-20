package luar

// Metamethods.

import (
	"fmt"
	"math"
	"math/cmplx"
	"reflect"

	"github.com/aarzilli/golua/lua"
)

func channel__index(L *lua.State) int {
	v, t := valueOfProxy(L, 1)
	name := L.ToString(2)
	switch name {
	case "recv":
		f := func(L *lua.State) int {
			val, ok := v.Recv()
			if ok {
				GoToLuaProxy(L, val)
			} else {
				L.PushNil()
			}
			return 1
		}
		L.PushGoFunction(f)
	case "send":
		f := func(L *lua.State) int {
			val := reflect.New(t.Elem())
			LuaToGo(L, 1, val.Interface())
			v.Send(val.Elem())
			return 0
		}
		L.PushGoFunction(f)
	case "close":
		f := func(L *lua.State) int {
			v.Close()
			return 0
		}
		L.PushGoFunction(f)
	default:
		pushGoMethod(L, name, v)
	}
	return 1
}

func complex__index(L *lua.State) int {
	v, _ := valueOfProxy(L, 1)
	name := L.ToString(2)
	switch name {
	case "real":
		L.PushNumber(real(v.Complex()))
	case "imag":
		L.PushNumber(imag(v.Complex()))
	default:
		pushGoMethod(L, name, v)
	}
	return 1
}

func interface__index(L *lua.State) int {
	v, _ := valueOfProxy(L, 1)
	name := L.ToString(2)
	pushGoMethod(L, name, v)
	return 1
}

func map__index(L *lua.State) int {
	v, t := valueOfProxy(L, 1)
	key := reflect.New(t.Key())
	LuaToGo(L, 2, key.Interface()) // TODO: Error check? For all mm?
	key = key.Elem()
	val := v.MapIndex(key)
	if val.IsValid() {
		GoToLuaProxy(L, val)
		return 1
	} else if key.Kind() == reflect.String {
		name := key.String()

		// From 'pushGoMethod':
		val := v.MethodByName(name)
		if !val.IsValid() {
			T := v.Type()
			// Could not resolve this method. Perhaps it's defined on the pointer?
			if T.Kind() != reflect.Ptr {
				if v.CanAddr() {
					v = v.Addr()
				} else {
					vp := reflect.New(T)
					vp.Elem().Set(v)
					v = vp
				}
			}
			val = v.MethodByName(name)
			// Unlike 'pushGoMethod', do not panic.
			if !val.IsValid() {
				L.PushNil()
				return 1
			}
		}
		GoToLua(L, val)
		return 1
	}
	return 0
}

func map__ipairs(L *lua.State) int {
	v, _ := valueOfProxy(L, 1)
	keys := v.MapKeys()
	intKeys := map[uint64]reflect.Value{}

	// Filter integer keys.
	for _, k := range keys {
		if k.Kind() == reflect.Interface {
			k = k.Elem()
		}
		switch unsizedKind(k) {
		case reflect.Int64:
			i := k.Int()
			if i > 0 {
				intKeys[uint64(i)] = k
			}
		case reflect.Uint64:
			intKeys[k.Uint()] = k
		}
	}

	idx := uint64(0)
	iter := func(L *lua.State) int {
		idx++
		if _, ok := intKeys[idx]; !ok {
			L.PushNil()
			return 1
		}
		GoToLuaProxy(L, idx)
		val := v.MapIndex(intKeys[idx])
		GoToLuaProxy(L, val)
		return 2
	}
	L.PushGoFunction(iter)
	return 1
}

func map__newindex(L *lua.State) int {
	v, t := valueOfProxy(L, 1)
	key := reflect.New(t.Key())
	LuaToGo(L, 2, key.Interface())
	key = key.Elem()
	val := reflect.New(t.Elem())
	LuaToGo(L, 3, val.Interface())
	val = val.Elem()
	v.SetMapIndex(key, val)
	return 0
}

func map__pairs(L *lua.State) int {
	v, _ := valueOfProxy(L, 1)
	keys := v.MapKeys()
	idx := -1
	n := v.Len()
	iter := func(L *lua.State) int {
		idx++
		if idx == n {
			L.PushNil()
			return 1
		}
		GoToLuaProxy(L, keys[idx])
		val := v.MapIndex(keys[idx])
		GoToLuaProxy(L, val)
		return 2
	}
	L.PushGoFunction(iter)
	return 1
}

func number__add(L *lua.State) int {
	v1, t1 := luaToGoValue(L, 1)
	v2, t2 := luaToGoValue(L, 2)
	var result interface{}
	switch commonKind(v1, v2) {
	case reflect.Uint64:
		result = v1.Uint() + v2.Uint()
	case reflect.Int64:
		result = v1.Int() + v2.Int()
	case reflect.Float64:
		result = valueToNumber(L, v1) + valueToNumber(L, v2)
	case reflect.Complex128:
		result = valueToComplex(L, v1) + valueToComplex(L, v2)
	}
	pushNumberValue(L, result, t1, t2)
	return 1
}

func number__div(L *lua.State) int {
	v1, t1 := luaToGoValue(L, 1)
	v2, t2 := luaToGoValue(L, 2)
	var result interface{}
	switch commonKind(v1, v2) {
	case reflect.Uint64:
		result = v1.Uint() / v2.Uint()
	case reflect.Int64:
		result = v1.Int() / v2.Int()
	case reflect.Float64:
		result = valueToNumber(L, v1) / valueToNumber(L, v2)
	case reflect.Complex128:
		result = valueToComplex(L, v1) / valueToComplex(L, v2)
	}
	pushNumberValue(L, result, t1, t2)
	return 1
}

func number__lt(L *lua.State) int {
	v1, _ := luaToGoValue(L, 1)
	v2, _ := luaToGoValue(L, 2)
	switch commonKind(v1, v2) {
	case reflect.Uint64:
		L.PushBoolean(v1.Uint() < v2.Uint())
	case reflect.Int64:
		L.PushBoolean(v1.Int() < v2.Int())
	case reflect.Float64:
		L.PushBoolean(valueToNumber(L, v1) < valueToNumber(L, v2))
	}
	return 1
}

func number__mod(L *lua.State) int {
	v1, t1 := luaToGoValue(L, 1)
	v2, t2 := luaToGoValue(L, 2)
	var result interface{}
	switch commonKind(v1, v2) {
	case reflect.Uint64:
		result = v1.Uint() % v2.Uint()
	case reflect.Int64:
		result = v1.Int() % v2.Int()
	case reflect.Float64:
		result = math.Mod(valueToNumber(L, v1), valueToNumber(L, v2))
	}
	pushNumberValue(L, result, t1, t2)
	return 1
}

func number__mul(L *lua.State) int {
	v1, t1 := luaToGoValue(L, 1)
	v2, t2 := luaToGoValue(L, 2)
	var result interface{}
	switch commonKind(v1, v2) {
	case reflect.Uint64:
		result = v1.Uint() * v2.Uint()
	case reflect.Int64:
		result = v1.Int() * v2.Int()
	case reflect.Float64:
		result = valueToNumber(L, v1) * valueToNumber(L, v2)
	case reflect.Complex128:
		result = valueToComplex(L, v1) * valueToComplex(L, v2)
	}
	pushNumberValue(L, result, t1, t2)
	return 1
}

func number__pow(L *lua.State) int {
	v1, t1 := luaToGoValue(L, 1)
	v2, t2 := luaToGoValue(L, 2)
	var result interface{}
	switch commonKind(v1, v2) {
	case reflect.Uint64:
		result = math.Pow(float64(v1.Uint()), float64(v2.Uint()))
	case reflect.Int64:
		result = math.Pow(float64(v1.Int()), float64(v2.Int()))
	case reflect.Float64:
		result = math.Pow(valueToNumber(L, v1), valueToNumber(L, v2))
	case reflect.Complex128:
		result = cmplx.Pow(valueToComplex(L, v1), valueToComplex(L, v2))
	}
	pushNumberValue(L, result, t1, t2)
	return 1
}

func number__sub(L *lua.State) int {
	v1, t1 := luaToGoValue(L, 1)
	v2, t2 := luaToGoValue(L, 2)
	var result interface{}
	switch commonKind(v1, v2) {
	case reflect.Uint64:
		result = v1.Uint() - v2.Uint()
	case reflect.Int64:
		result = v1.Int() - v2.Int()
	case reflect.Float64:
		result = valueToNumber(L, v1) - valueToNumber(L, v2)
	case reflect.Complex128:
		result = valueToComplex(L, v1) - valueToComplex(L, v2)
	}
	pushNumberValue(L, result, t1, t2)
	return 1
}

func number__unm(L *lua.State) int {
	v1, t1 := luaToGoValue(L, 1)
	var result interface{}
	switch unsizedKind(v1) {
	case reflect.Uint64:
		result = -v1.Uint()
	case reflect.Int64:
		result = -v1.Int()
	case reflect.Float64, reflect.String:
		result = -valueToNumber(L, v1)
	case reflect.Complex128:
		result = -v1.Complex()
	}
	v := reflect.ValueOf(result)
	if unsizedKind(v1) == reflect.Complex128 {
		makeValueProxy(L, v.Convert(t1), cComplexMeta)
	} else if isNewType(t1) {
		makeValueProxy(L, v.Convert(t1), cNumberMeta)
	} else {
		L.PushNumber(v.Float())
	}
	return 1
}

// From Lua's specs: "A metamethod only is selected when both objects being
// compared have the same type and the same metamethod for the selected
// operation." Thus both arguments must be proxies for this function to be
// called. No need to check for type equality: Go's "==" operator will do it for
// us.
func proxy__eq(L *lua.State) int {
	var v1 interface{}
	LuaToGo(L, 1, &v1)
	var v2 interface{}
	LuaToGo(L, 2, &v2)
	L.PushBoolean(v1 == v2)
	return 1
}

func proxy__gc(L *lua.State) int {
	vp := (*valueProxy)(L.ToUserdata(1))
	proxymu.Lock()
	delete(proxyMap, vp)
	proxymu.Unlock()
	return 0
}

func proxy__tostring(L *lua.State) int {
	v, _ := valueOfProxy(L, 1)
	L.PushString(fmt.Sprintf("%v", v))
	return 1
}

func slice__index(L *lua.State) int {
	v, _ := valueOfProxy(L, 1)
	for v.Kind() == reflect.Ptr {
		// For arrays.
		v = v.Elem()
	}
	if L.IsNumber(2) {
		idx := L.ToInteger(2)
		if idx < 1 || idx > v.Len() {
			RaiseError(L, "slice/array get: index out of range")
		}
		v := v.Index(idx - 1)
		GoToLuaProxy(L, v)
	} else if L.IsString(2) {
		name := L.ToString(2)
		if v.Kind() == reflect.Array {
			pushGoMethod(L, name, v)
			return 1
		}
		switch name {
		case "append":
			f := func(L *lua.State) int {
				narg := L.GetTop()
				args := []reflect.Value{}
				for i := 1; i <= narg; i++ {
					elem := reflect.New(v.Type().Elem())
					LuaToGo(L, i, elem.Interface())
					args = append(args, elem.Elem())
				}
				newslice := reflect.Append(v, args...)
				makeValueProxy(L, newslice, cSliceMeta)
				return 1
			}
			L.PushGoFunction(f)
		case "cap":
			L.PushInteger(int64(v.Cap()))
		case "sub":
			f := func(L *lua.State) int {
				i1, i2 := L.ToInteger(1), L.ToInteger(2)
				newslice := v.Slice(i1-1, i2)
				makeValueProxy(L, newslice, cSliceMeta)
				return 1
			}
			L.PushGoFunction(f)
		default:
			pushGoMethod(L, name, v)
		}
	} else {
		RaiseError(L, "slice/array requires integer index")
	}
	return 1
}

func slice__ipairs(L *lua.State) int {
	v, _ := valueOfProxy(L, 1)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	n := v.Len()
	idx := -1
	iter := func(L *lua.State) int {
		idx++
		if idx == n {
			L.PushNil()
			return 1
		}
		GoToLuaProxy(L, idx+1) // report as 1-based index
		val := v.Index(idx)
		GoToLuaProxy(L, val)
		return 2
	}
	L.PushGoFunction(iter)
	return 1
}

func slice__newindex(L *lua.State) int {
	v, t := valueOfProxy(L, 1)
	for v.Kind() == reflect.Ptr {
		// For arrays.
		v = v.Elem()
		t = t.Elem()
	}
	idx := L.ToInteger(2)
	val := reflect.New(t.Elem())
	LuaToGo(L, 3, val.Interface())
	val = val.Elem()
	if idx < 1 || idx > v.Len() {
		RaiseError(L, "slice/array set: index out of range")
	}
	v.Index(idx - 1).Set(val)
	return 0
}

func slicemap__len(L *lua.State) int {
	v, _ := valueOfProxy(L, 1)
	for v.Kind() == reflect.Ptr {
		// For arrays.
		v = v.Elem()
	}
	L.PushInteger(int64(v.Len()))
	return 1
}

// Lua accepts concatenation with string and number.
func string__concat(L *lua.State) int {
	v1, t1 := luaToGoValue(L, 1)
	v2, t2 := luaToGoValue(L, 2)
	s1 := valueToString(L, v1)
	s2 := valueToString(L, v2)
	result := s1 + s2

	if t1 == t2 || isPredeclaredType(t2) {
		v := reflect.ValueOf(result)
		makeValueProxy(L, v.Convert(t1), cStringMeta)
	} else if isPredeclaredType(t1) {
		v := reflect.ValueOf(result)
		makeValueProxy(L, v.Convert(t2), cStringMeta)
	} else {
		L.PushString(result)
	}

	return 1
}
func string__index(L *lua.State) int {
	v, _ := valueOfProxy(L, 1)
	name := L.ToString(2)
	if name == "sub" {
		f := func(L *lua.State) int {
			i1, i2 := L.ToInteger(1), L.ToInteger(2)
			vn := v.Slice(i1-1, i2)
			makeValueProxy(L, vn, cStringMeta)
			return 1
		}
		L.PushGoFunction(f)

	} else {
		pushGoMethod(L, name, v)
	}
	return 1
}

func string__ipairs(L *lua.State) int {
	v, _ := valueOfProxy(L, 1)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	r := []rune(v.String())
	n := len(r)
	idx := -1
	iter := func(L *lua.State) int {
		idx++
		if idx == n {
			L.PushNil()
			return 1
		}
		GoToLuaProxy(L, idx+1) // report as 1-based index
		GoToLuaProxy(L, string(r[idx]))
		return 2
	}
	L.PushGoFunction(iter)
	return 1
}

func string__len(L *lua.State) int {
	v1, _ := luaToGoValue(L, 1)
	L.PushInteger(int64(v1.Len()))
	return 1
}

func string__lt(L *lua.State) int {
	v1, _ := luaToGoValue(L, 1)
	v2, _ := luaToGoValue(L, 2)
	L.PushBoolean(v1.String() < v2.String())
	return 1
}

func struct__index(L *lua.State) int {
	v, t := valueOfProxy(L, 1)
	name := L.ToString(2)
	vp := v
	if t.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	field := v.FieldByName(name)
	if !field.IsValid() || !field.CanSet() {
		// No such exported field, try for method.
		pushGoMethod(L, name, vp)
	} else {
		if isPointerToPrimitive(field) {
			GoToLuaProxy(L, field.Elem())
		} else {
			GoToLuaProxy(L, field)
		}
	}
	return 1
}

func struct__newindex(L *lua.State) int {
	v, t := valueOfProxy(L, 1)
	name := L.ToString(2)
	if t.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	field := v.FieldByName(name)
	assertValid(L, field, v, name, "field")
	val := reflect.New(field.Type())
	LuaToGo(L, 3, val.Interface())
	val = val.Elem()
	if isPointerToPrimitive(field) {
		field.Elem().Set(val)
	} else {
		field.Set(val)
	}
	return 0
}