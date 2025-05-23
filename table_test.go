package lua

import (
	"testing"
)

func TestTableNewLTable(t *testing.T) {
	tbl := newLTable(-1, -2)
	errorIfNotEqual(t, 0, cap(tbl.array))

	tbl = newLTable(10, 9)
	errorIfNotEqual(t, 10, cap(tbl.array))
}

func TestTableLen(t *testing.T) {
	tbl := newLTable(0, 0)
	tbl.RawSetInt(10, LNil)
	tbl.RawSetInt(9, LNumber(10))
	tbl.RawSetInt(8, LNil)
	tbl.RawSetInt(7, LNumber(10))
	errorIfNotEqual(t, 9, tbl.Len())

	tbl = newLTable(0, 0)
	tbl.Append(LTrue)
	tbl.Append(LTrue)
	tbl.Append(LTrue)
	errorIfNotEqual(t, 3, tbl.Len())
}

func TestTableLenType(t *testing.T) {
	L := NewState(Options{})
	err := L.DoString(`
        mt = {
            __index = mt,
            __len = function (self)
                return {hello = "world"}
            end
        }

        v = {}
        v.__index = v

        setmetatable(v, mt)

        assert(#v ~= 0, "#v should return a table reference in this case")

        print(#v)
    `)
	if err != nil {
		t.Error(err)
	}
}

func TestTableAppend(t *testing.T) {
	tbl := newLTable(0, 0)
	tbl.RawSetInt(1, LNumber(1))
	tbl.RawSetInt(2, LNumber(2))
	tbl.RawSetInt(3, LNumber(3))
	errorIfNotEqual(t, 3, tbl.Len())

	tbl.RawSetInt(1, LNil)
	tbl.RawSetInt(2, LNil)
	errorIfNotEqual(t, 3, tbl.Len())

	tbl.Append(LNumber(4))
	errorIfNotEqual(t, 4, tbl.Len())

	tbl.RawSetInt(3, LNil)
	tbl.RawSetInt(4, LNil)
	errorIfNotEqual(t, 0, tbl.Len())

	tbl.Append(LNumber(5))
	errorIfNotEqual(t, 1, tbl.Len())
}

func TestTableInsert(t *testing.T) {
	tbl := newLTable(0, 0)
	tbl.Append(LTrue)
	tbl.Append(LTrue)
	tbl.Append(LTrue)

	tbl.Insert(5, LFalse)
	errorIfNotEqual(t, LFalse, tbl.RawGetInt(5))
	errorIfNotEqual(t, 5, tbl.Len())

	tbl.Insert(-10, LFalse)
	errorIfNotEqual(t, LFalse, tbl.RawGet(LNumber(-10)))
	errorIfNotEqual(t, 5, tbl.Len())

	tbl = newLTable(0, 0)
	tbl.Append(LNumber(1))
	tbl.Append(LNumber(2))
	tbl.Append(LNumber(3))
	tbl.Insert(1, LNumber(10))
	errorIfNotEqual(t, LNumber(10), tbl.RawGetInt(1))
	errorIfNotEqual(t, LNumber(1), tbl.RawGetInt(2))
	errorIfNotEqual(t, LNumber(2), tbl.RawGetInt(3))
	errorIfNotEqual(t, LNumber(3), tbl.RawGetInt(4))
	errorIfNotEqual(t, 4, tbl.Len())

	tbl = newLTable(0, 0)
	tbl.Insert(5, LNumber(10))
	errorIfNotEqual(t, LNumber(10), tbl.RawGetInt(5))

}

func TestTableMaxN(t *testing.T) {
	tbl := newLTable(0, 0)
	tbl.Append(LTrue)
	tbl.Append(LTrue)
	tbl.Append(LTrue)
	errorIfNotEqual(t, 3, tbl.MaxN())

	tbl = newLTable(0, 0)
	errorIfNotEqual(t, 0, tbl.MaxN())

	tbl = newLTable(10, 0)
	errorIfNotEqual(t, 0, tbl.MaxN())
}

func TestTableRemove(t *testing.T) {
	tbl := newLTable(0, 0)
	errorIfNotEqual(t, LNil, tbl.Remove(10))
	tbl.Append(LTrue)
	errorIfNotEqual(t, LNil, tbl.Remove(10))

	tbl.Append(LFalse)
	tbl.Append(LTrue)
	errorIfNotEqual(t, LFalse, tbl.Remove(2))
	errorIfNotEqual(t, 2, tbl.MaxN())
	tbl.Append(LFalse)
	errorIfNotEqual(t, LFalse, tbl.Remove(-1))
	errorIfNotEqual(t, 2, tbl.MaxN())

}

func TestTableRawSetInt(t *testing.T) {
	tbl := newLTable(0, 0)
	tbl.RawSetInt(MaxArrayIndex+1, LTrue)
	errorIfNotEqual(t, 0, tbl.MaxN())
	errorIfNotEqual(t, LTrue, tbl.RawGet(LNumber(MaxArrayIndex+1)))

	tbl.RawSetInt(1, LTrue)
	tbl.RawSetInt(3, LTrue)
	errorIfNotEqual(t, 3, tbl.MaxN())
	errorIfNotEqual(t, LTrue, tbl.RawGetInt(1))
	errorIfNotEqual(t, LNil, tbl.RawGetInt(2))
	errorIfNotEqual(t, LTrue, tbl.RawGetInt(3))
	tbl.RawSetInt(2, LTrue)
	errorIfNotEqual(t, LTrue, tbl.RawGetInt(1))
	errorIfNotEqual(t, LTrue, tbl.RawGetInt(2))
	errorIfNotEqual(t, LTrue, tbl.RawGetInt(3))
}

func TestTableRawSetH(t *testing.T) {
	tbl := newLTable(0, 0)
	tbl.RawSetH(LString("key"), LTrue)
	tbl.RawSetH(LString("key"), LNil)
	_, found := tbl.dict[LString("key")]
	errorIfNotEqual(t, false, found)

	tbl.RawSetH(LTrue, LTrue)
	tbl.RawSetH(LTrue, LNil)
	_, foundb := tbl.dict[LTrue]
	errorIfNotEqual(t, false, foundb)
}

func TestTableRawGetH(t *testing.T) {
	tbl := newLTable(0, 0)
	errorIfNotEqual(t, LNil, tbl.RawGetH(LNumber(1)))
	errorIfNotEqual(t, LNil, tbl.RawGetH(LString("key0")))
	tbl.RawSetH(LString("key0"), LTrue)
	tbl.RawSetH(LString("key1"), LFalse)
	tbl.RawSetH(LNumber(1), LTrue)
	errorIfNotEqual(t, LTrue, tbl.RawGetH(LString("key0")))
	errorIfNotEqual(t, LTrue, tbl.RawGetH(LNumber(1)))
	errorIfNotEqual(t, LNil, tbl.RawGetH(LString("notexist")))
	errorIfNotEqual(t, LNil, tbl.RawGetH(LTrue))
}

func TestTableForEach(t *testing.T) {
	tbl := newLTable(0, 0)
	tbl.Append(LNumber(1))
	tbl.Append(LNumber(2))
	tbl.Append(LNumber(3))
	tbl.Append(LNil)
	tbl.Append(LNumber(5))

	tbl.RawSetH(LString("a"), LString("a"))
	tbl.RawSetH(LString("b"), LString("b"))
	tbl.RawSetH(LString("c"), LString("c"))

	tbl.RawSetH(LTrue, LString("true"))
	tbl.RawSetH(LFalse, LString("false"))

	tbl.ForEach(func(key, value LValue) {
		switch k := key.(type) {
		case LBool:
			switch bool(k) {
			case true:
				errorIfNotEqual(t, LString("true"), value)
			case false:
				errorIfNotEqual(t, LString("false"), value)
			default:
				t.Fail()
			}
		case LNumber:
			switch int(k) {
			case 1:
				errorIfNotEqual(t, LNumber(1), value)
			case 2:
				errorIfNotEqual(t, LNumber(2), value)
			case 3:
				errorIfNotEqual(t, LNumber(3), value)
			case 4:
				errorIfNotEqual(t, LNumber(5), value)
			default:
				t.Fail()
			}
		case LString:
			switch string(k) {
			case "a":
				errorIfNotEqual(t, LString("a"), value)
			case "b":
				errorIfNotEqual(t, LString("b"), value)
			case "c":
				errorIfNotEqual(t, LString("c"), value)
			default:
				t.Fail()
			}
		}
	})
}

func TestTableNext(t *testing.T) {
	L := NewState(Options{})
	err := L.DoString(`
        table = {
			[1] = {
 				123
			},
            [1001]=
			  {
			  	servername=''
			  },
            [1002]=
			  {
				servername=''
			  }
        }

		for k, v in pairs(table) do
			print(k, v)
		end

		print("table.len: " .. #table)
    `)

	if err != nil {
		t.Error(err)
	}
}

func TestTableSort(t *testing.T) {
	L := NewState(Options{})
	err := L.DoString(`
	function test()
        a = {
			4, 5, 6
        }

		print("a.len " .. table.getn(a))

		table.sort(a)

		table.insert(a, 3)
		table.insert(a, 2)
		table.insert(a, 1)

		print("a.len " .. table.getn(a))

		table.sort(a)

		for k, v in pairs(a) do
			print(k, v)
		end

		print("after sort: ")
		print("a.len " .. table.getn(a))

		table.insert(a, 2)
		print("a.len " .. table.getn(a))

		table.remove(a, 3)
		print("a.len " .. table.getn(a))
		a[#a] = nil

		print("a.len " .. table.getn(a))
		for k, v in pairs(a) do
			print(k, v)
		end

		table.sort(a)
		print ("======================")

		for k, v in pairs(a) do
			print(k, v)
		end
		print("a.len: " .. table.getn(a))
	end
	test()

    `)

	if err != nil {
		t.Error(err)
	}
}

func TestTableUnpack(t *testing.T) {
	L := NewState(Options{})
	err := L.DoString(`

	__print = __print or print

	function _print(...)
		local t = {...}
		for _, arg in ipairs(t or {}) do
			if type(arg) == TYPE_STR or type(arg) == TYPE_NUM then
				__print(arg)
			elseif type(arg) == TYPE_BOOL or type(arg) == TYPE_FUN then
				__print(tostring(arg))
			elseif type(arg) == TYPE_TAB then
				PrintTable(arg)
			else
				__print(arg)
			end
		end
	end

	local printed = {}
	local function _PrintTable( t, space )
		if printed[t] then
			return
		end
		printed[t] = 1
	
		if( space == nil ) then space = "  " end
		if( t == nil )then
			_print( space.."nil" )
			return
		end
		if( type(t) ~= "table" )then
			_print( space .. " expected table, got " .. type(t))
			return
		end
		_print( space.."{" )
		for k,v in pairs(t) do
			if( type(v) ~= "table" )then
				if( type(v) == "string" )then
					if( type(k) == "number" )then
						_print( space.."["..k.."]='"..tostring(v).."'" )
					else
						_print( space..k.."='"..tostring(v).."'" )
					end
				else
					if( type(k) == "number" )then
						_print( space.."["..k.."]="..tostring(v) )
					else
						_print( space..k.."="..tostring(v) )
					end
				end
			else
				if( type(k) == "number" )then
					_print( space.."["..k.."]=" )
				else
					_print( space..k.."=" )
				end
				spaceNext = space.."  "
				PrintTable( v, spaceNext )
			end
		end
		_print( space.."}" )
	end

	function PrintTable(t,space)
		table.clear(printed)
		_PrintTable( t, space )
	end

	function test(...)
		local a = {...}
		PrintTable({unpack(a)})
	end

	test({}, 1, "1231231", { "a", "b", "c", "d", "e", "f", "g", "h", "i" })

    `)

	if err != nil {
		t.Error(err)
	}
}
