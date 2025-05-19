package lua

const defaultArrayCap = 32
const defaultHashCap = 32
const maxBits = 32

func NewLValueArraySorter(L *LState, Fn *LFunction, Values []LValue) lValueArraySorter {
	sorter := lValueArraySorter{L, Fn, RebuildValues(Values)}
	return sorter
}

type lValueArraySorter struct {
	L      *LState
	Fn     *LFunction
	Values []LValue
}

func (lv lValueArraySorter) Len() int {
	return len(lv.Values)
}

func (lv lValueArraySorter) Swap(i, j int) {
	lv.Values[i], lv.Values[j] = lv.Values[j], lv.Values[i]
}

func (lv lValueArraySorter) Less(i, j int) bool {
	if lv.Fn != nil {
		lv.L.Push(lv.Fn)
		lv.L.Push(lv.Values[i])
		lv.L.Push(lv.Values[j])
		lv.L.Call(2, 1)
		return LVAsBool(lv.L.reg.Pop())
	}
	return lessThan(lv.L, lv.Values[i], lv.Values[j])
}

func RebuildValues(array []LValue) []LValue {
	cnt := 0
	for _, v := range array {
		if v != nil && v != LNil {
			cnt += 1
		}
	}

	newArray := make([]LValue, cnt)
	index := 0
	for _, v := range array {
		if v != nil && v != LNil {
			newArray[index] = v
			index++
		}
	}

	return newArray
}

func newLTable(acap int, hcap int) *LTable {
	if acap < 0 {
		acap = 0
	}
	if hcap < 0 {
		hcap = 0
	}
	tb := &LTable{}
	tb.Metatable = LNil
	if acap != 0 {
		tb.array = make([]LValue, 0, acap)
	}
	if hcap != 0 {
		tb.strdict = make(map[string]LValue, hcap)
	}
	return tb
}

func (tb *LTable) createNewArray(cap int) []LValue {
	ret := make([]LValue, cap)
	for i := 0; i < cap; i++ {
		ret[i] = LNil
	}
	return ret
}

// Len returns length of this LTable without using __len.
func (tb *LTable) Len() int {
	if tb.array == nil {
		return 0
	}
	var prev LValue = LNil
	for i := len(tb.array) - 1; i >= 0; i-- {
		v := tb.array[i]
		if prev == LNil && v != LNil {
			return i + 1
		}
		prev = v
	}
	return 0
}

// Append appends a given LValue to this LTable.
func (tb *LTable) Append(value LValue) {
	if value == LNil {
		return
	}
	if tb.array == nil {
		tb.array = tb.createNewArray(defaultArrayCap)
	}

	tailIndex := len(tb.array) - 1
	if len(tb.array) == 0 || tb.array[tailIndex] != LNil {
		tb.rehash(LNumber(len(tb.array)))
		tb.array[tailIndex+1] = value
	} else {
		// 找到最后一个不为空的位置设置值
		i := len(tb.array) - 2
		for ; i >= 0; i-- {
			if tb.array[i] != LNil {
				break
			}
		}
		tb.array[i+1] = value
	}
}

// Insert inserts a given LValue at position `i` in this table.
func (tb *LTable) Insert(i int, value LValue) {
	if tb.array == nil {
		tb.array = tb.createNewArray(defaultArrayCap)
	}
	if i > len(tb.array) {
		tb.RawSetInt(i, value)
		return
	}
	if i <= 0 {
		tb.RawSet(LNumber(i), value)
		return
	}
	i -= 1
	tb.array = append(tb.array, LNil)
	copy(tb.array[i+1:], tb.array[i:])
	tb.array[i] = value
}

// MaxN returns a maximum number key that nil value does not exist before it.
func (tb *LTable) MaxN() int {
	if tb.array == nil {
		return 0
	}
	for i := len(tb.array) - 1; i >= 0; i-- {
		if tb.array[i] != LNil {
			return i + 1
		}
	}
	return 0
}

// Remove removes from this table the element at a given position.
func (tb *LTable) Remove(pos int) LValue {
	if tb.array == nil {
		return LNil
	}
	larray := len(tb.array)
	if larray == 0 {
		return LNil
	}
	i := pos - 1
	oldval := LNil
	switch {
	case i >= larray:
		// TODO tb.keys and tb.k2i should also be removed
		// new feature 这里要移除在hash里的数字键...
		deleteIndex := -1
		for i, realKey := range tb.keys {
			if LNumber(pos) == realKey {
				deleteIndex = i
				tb.keys = append(tb.keys[:i], tb.keys[i+1:]...)
				break
			}
		}
		delete(tb.k2i, LNumber(pos))
		for realKey, index := range tb.k2i {
			if index > deleteIndex {
				tb.k2i[realKey] = tb.k2i[realKey] - 1
			}
		}
		// end ...
		delete(tb.dict, LNumber(pos))
		// nothing to do
	case i == larray-1 || i < 0:
		oldval = tb.array[larray-1]
		tb.array = tb.array[:larray-1]
	default:
		oldval = tb.array[i]
		copy(tb.array[i:], tb.array[i+1:])
		tb.array[larray-1] = nil
		tb.array = tb.array[:larray-1]
	}
	return oldval
}

// RawSet sets a given LValue to a given index without the __newindex metamethod.
// It is recommended to use `RawSetString` or `RawSetInt` for performance
// if you already know the given LValue is a string or number.
func (tb *LTable) RawSet(key LValue, value LValue) {
	switch v := key.(type) {
	case LNumber:
		if isArrayKey(v) && isArrayKeyEnhance(v, tb) {
			if value == LNil {
				if v > LNumber(tb.Len()) {
					return
				} else if v == LNumber(tb.Len()) {
					tb.Remove(int(v))
					return
				}
			}

			if tb.array == nil {
				tb.array = tb.createNewArray(defaultArrayCap)
			}
			index := int(v) - 1
			alen := len(tb.array)

			if index >= alen {
				// 需要扩展数组
				tb.resize(index+1, 0)
			}

			tb.array[index] = value
			return
		}
	case LString:
		tb.RawSetString(string(v), value)
		return
	}

	tb.RawSetH(key, value)
}

// RawSetInt sets a given LValue at a position `key` without the __newindex metamethod.
//func (tb *LTable) RawSetInt(key int, value LValue) {
//	if key < 1 || !isArrayKeyEnhance(LNumber(key), tb) || key >= MaxArrayIndex {
//		tb.RawSetH(LNumber(key), value)
//		return
//	}
//	if tb.array == nil {
//		tb.array = make([]LValue, 0, 32)
//	}
//	index := key - 1
//	alen := len(tb.array)
//	switch {
//	case index == alen:
//		tb.array = append(tb.array, value)
//	case index > alen:
//		for i := 0; i < (index - alen); i++ {
//			tb.array = append(tb.array, LNil)
//		}
//		tb.array = append(tb.array, value)
//	case index < alen:
//		tb.array[index] = value
//	}
//}

// RawSetInt sets a given LValue at a position `key` without the __newindex metamethod.
func (tb *LTable) RawSetInt(key int, value LValue) {
	if key < 1 || !isArrayKeyEnhance(LNumber(key), tb) || key >= MaxArrayIndex {
		tb.RawSetH(LNumber(key), value)
		return
	}
	if tb.array == nil {
		tb.array = tb.createNewArray(defaultArrayCap)
	}
	index := key - 1
	alen := len(tb.array)

	if index >= alen {
		tb.resize(index+1, 0)
	}

	// 设置值
	tb.array[index] = value
}

// RawSetString sets a given LValue to a given string index without the __newindex metamethod.
func (tb *LTable) RawSetString(key string, value LValue) {
	if tb.strdict == nil {
		tb.strdict = make(map[string]LValue, defaultHashCap)
	}
	if tb.keys == nil {
		tb.keys = []LValue{}
		tb.k2i = map[LValue]int{}
	}

	if value == LNil {
		// TODO tb.keys and tb.k2i should also be removed
		delete(tb.strdict, key)
	} else {
		tb.strdict[key] = value
		lkey := LString(key)
		if _, ok := tb.k2i[lkey]; !ok {
			tb.k2i[lkey] = len(tb.keys)
			tb.keys = append(tb.keys, lkey)
		}
	}
}

// RawSetH sets a given LValue to a given index without the __newindex metamethod.
func (tb *LTable) RawSetH(key LValue, value LValue) {
	if s, ok := key.(LString); ok {
		tb.RawSetString(string(s), value)
		return
	}
	if tb.dict == nil {
		tb.dict = make(map[LValue]LValue, len(tb.strdict))
	}
	if tb.keys == nil {
		tb.keys = []LValue{}
		tb.k2i = map[LValue]int{}
	}

	if value == LNil {
		// TODO tb.keys and tb.k2i should also be removed
		// new feature ...
		deleteIndex := -1
		for i, realKey := range tb.keys {
			if key == realKey {
				deleteIndex = i
				tb.keys = append(tb.keys[:i], tb.keys[i+1:]...)
				break
			}
		}
		delete(tb.k2i, key)
		for realKey, index := range tb.k2i {
			if index > deleteIndex {
				tb.k2i[realKey] = tb.k2i[realKey] - 1
			}
		}
		// end ...
		delete(tb.dict, key)
	} else {
		tb.dict[key] = value
		if _, ok := tb.k2i[key]; !ok {
			tb.k2i[key] = len(tb.keys)
			tb.keys = append(tb.keys, key)
		}
	}
}

// RawGet returns an LValue associated with a given key without __index metamethod.
func (tb *LTable) RawGet(key LValue) LValue {
	switch v := key.(type) {
	case LNumber:
		if isArrayKey(v) && isArrayKeyEnhance(v, tb) {
			if tb.array == nil {
				return LNil
			}
			index := int(v) - 1
			if index >= len(tb.array) {
				return LNil
			}
			return tb.array[index]
		}
	case LString:
		if tb.strdict == nil {
			return LNil
		}
		if ret, ok := tb.strdict[string(v)]; ok {
			return ret
		}
		return LNil
	}
	if tb.dict == nil {
		return LNil
	}
	if v, ok := tb.dict[key]; ok {
		return v
	}
	return LNil
}

// RawGetInt returns an LValue at position `key` without __index metamethod.
func (tb *LTable) RawGetInt(key int) LValue {
	if tb.array == nil {
		return LNil
	}
	index := int(key) - 1
	if !isArrayKeyEnhance(LNumber(key), tb) {
		return tb.RawGetH(LNumber(key))
	}
	if index >= len(tb.array) || index < 0 {
		return LNil
	}
	return tb.array[index]
}

// RawGetH returns an LValue associated with a given key without __index metamethod.
func (tb *LTable) RawGetH(key LValue) LValue {
	if s, sok := key.(LString); sok {
		if tb.strdict == nil {
			return LNil
		}
		if v, vok := tb.strdict[string(s)]; vok {
			return v
		}
		return LNil
	}
	if tb.dict == nil {
		return LNil
	}
	if v, ok := tb.dict[key]; ok {
		return v
	}
	return LNil
}

// RawGetString returns an LValue associated with a given key without __index metamethod.
func (tb *LTable) RawGetString(key string) LValue {
	if tb.strdict == nil {
		return LNil
	}
	if v, vok := tb.strdict[string(key)]; vok {
		return v
	}
	return LNil
}

// ForEach iterates over this table of elements, yielding each in turn to a given function.
func (tb *LTable) ForEach(cb func(LValue, LValue)) {
	if tb.array != nil {
		for i, v := range tb.array {
			if v != LNil {
				cb(LNumber(i+1), v)
			}
		}
	}
	if tb.strdict != nil {
		for k, v := range tb.strdict {
			if v != LNil {
				cb(LString(k), v)
			}
		}
	}
	if tb.dict != nil {
		for k, v := range tb.dict {
			if v != LNil {
				cb(k, v)
			}
		}
	}
}

// Next This function is equivalent to lua_next ( http://www.lua.org/manual/5.1/manual.html#lua_next ).
func (tb *LTable) Next(key LValue) (LValue, LValue) {
	init := false
	if key == LNil {
		key = LNumber(0)
		init = true
		tb.pairsHashFlag = false
	}

	if !tb.pairsHashFlag && (init || key != LNumber(0)) {
		if kv, ok := key.(LNumber); ok && isInteger(kv) && int(kv) >= 0 && kv < LNumber(MaxArrayIndex) {
			index := int(kv)
			if tb.array != nil && isArrayKeyEnhance(LNumber(index), tb) {
				for ; index < len(tb.array); index++ {
					if v := tb.array[index]; v != LNil {
						return LNumber(index + 1), v
					}
				}
			}
			if tb.array == nil || index == len(tb.array) {
				if (tb.dict == nil || len(tb.dict) == 0) && (tb.strdict == nil || len(tb.strdict) == 0) {
					return LNil, LNil
				}

				tb.pairsHashFlag = true
				key = tb.keys[0]
				if v := tb.RawGetH(key); v != LNil {
					return key, v
				}
			}
		}
	}

	for i := tb.k2i[key] + 1; i < len(tb.keys); i++ {
		key := tb.keys[i]
		if v := tb.RawGetH(key); v != LNil {
			return key, v
		}
	}
	return LNil, LNil
}

// resize 调整表的结构并重新插入元素
func (t *LTable) resize(nasize, nhsize int) {
	// 保存旧的数组和哈希部分
	oldArray := t.array
	oldHash := t.dict

	// 创建新的数组部分，直接翻倍容量
	newCap := cap(oldArray) * 2
	if newCap < nasize {
		newCap = nasize
	}
	if newCap <= 0 {
		newCap = defaultArrayCap
	}

	if nhsize <= 0 {
		nhsize = len(t.dict)
	}

	// 创建新的数组部分，确保所有元素初始化为LNil
	newArray := t.createNewArray(newCap)

	// 创建新的哈希部分
	newHash := make(map[LValue]LValue, nhsize)

	// 重新构造k2i, keys
	t.k2i = map[LValue]int{}
	t.keys = []LValue{}

	// 重新插入数组部分的元素
	for i, v := range oldArray {
		if v != LNil {
			key := LNumber(i + 1)
			t.rawset(key, v, newArray, newHash)
		}
	}

	// 重新插入哈希部分的元素
	needRepeatResize := false
	for k, v := range oldHash {
		if v != LNil {
			isHash := t.rawset(k, v, newArray, newHash)

			if isHash {
				if _, ok := t.k2i[k]; !ok {
					t.k2i[k] = len(t.keys)
					t.keys = append(t.keys, k)
				}
			} else {
				// repeat resize action
				needRepeatResize = true
			}
		}
	}

	// 更新表的数组和哈希部分
	t.array = newArray
	t.dict = newHash

	if needRepeatResize {
		t.resize(nasize, len(newHash))
	}
}

// rawset 将键值对插入到表中的适当位置
func (t *LTable) rawset(key, value LValue, array []LValue, hash map[LValue]LValue) bool {
	if n, ok := key.(LNumber); ok {
		if isArrayIndex(n) {
			index := int(n) - 1
			if index >= 0 && index < len(array) {
				array[index] = value
				return false
			}
		}
	}

	hash[key] = value
	return true
}

// ceillog2 计算大于等于n的最小2的幂的对数
func ceillog2(n int) int {
	log := 0
	for 1<<log < n {
		log++
	}
	return log
}

// countArrayElements 统计数组部分中的元素分布
func (t *LTable) countArrayElements(nums []int) int {
	ause := 0
	for i := 0; i < len(t.array); i++ {
		if t.array[i] != LNil {
			lg := ceillog2(i + 1)
			nums[lg]++
			ause++
		}
	}
	return ause
}

// countHashElements 统计哈希部分中可以作为数组索引的元素
func (t *LTable) countHashElements(nums []int, pnasize *int) int {
	totaluse := 0
	ause := 0

	for k, v := range t.dict {
		if v != LNil {
			if n, ok := k.(LNumber); ok {
				if isArrayIndex(n) {
					idx := int(n)
					if idx > 0 && idx <= maxBits {
						nums[ceillog2(idx)]++
						ause++
					}
				}
			}
			totaluse++
		}
	}

	*pnasize += ause
	return totaluse
}

// computeSizes 计算最优的数组部分大小
func computeSizes(nums []int, narray *int) int {
	a := 0  // 小于2^i的元素数量
	na := 0 // 将进入数组部分的元素数量
	n := 0  // 数组部分的最优大小

	for i, twotoi := 0, 1; twotoi/2 < *narray; i, twotoi = i+1, twotoi*2 {
		if nums[i] > 0 {
			a += nums[i]
			if a > twotoi/2 { // 超过一半的位置被使用？
				n = twotoi // 当前的最优大小
				na = a     // 所有小于n的元素将进入数组部分
			}
		}
		if a == *narray {
			break // 所有元素已计数
		}
	}

	*narray = n
	return na
}

// isArrayIndex 检查一个数字是否可以作为数组索引
func isArrayIndex(n LNumber) bool {
	v := float64(n)
	return v == float64(int(v)) && v > 0
}

// rehash 重新平衡表的数组部分和哈希部分
// 简化版本：直接将数组容量翻倍
func (t *LTable) rehash(key LValue) {
	// 获取需要的数组大小
	var nasize int
	if k, ok := key.(LNumber); ok && isArrayIndex(k) {
		nasize = int(k)
	} else {
		nasize = len(t.array) + 1
	}

	// 调整表结构，数组容量会在resize函数中翻倍
	t.resize(nasize, len(t.dict))
}
