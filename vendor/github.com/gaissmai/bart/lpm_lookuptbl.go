package bart

import "github.com/gaissmai/bart/internal/bitset"

// lpmLookupTbl is the backtracking sequence in the complete binary tree as bitstring.
//
//	for idx := 1; idx > 0; idx >>= 1 { b.Set(idx) }
//
// allows a one shot bitset intersection algorithm:
//
//	func (n *node[V]) lpmTest(idx uint) bool {
//		return n.prefixes.IntersectsAny(lpmLookupTbl[idx])
//	}
//
// instead of a sequence of single bitset tests:
//
//	func (n *node[V]) lpmTest(idx uint) bool {
//		for ; idx > 0; idx >>= 1 {
//			if n.prefixes.Test(idx) {
//				return true
//			}
//		}
//		return false
//	}
var lpmLookupTbl = [512]bitset.BitSet{
	/* idx:   0 */ {}, // invalid
	/* idx:   1 */ {0x2}, // 0b0000_0010
	/* idx:   2 */ {0x6}, // 0b0000_0110
	/* idx:   3 */ {0xa}, // 0b0000_1010
	/* idx:   4 */ {0x16}, // ...
	/* idx:   5 */ {0x26},
	/* idx:   6 */ {0x4a},
	/* idx:   7 */ {0x8a},
	/* idx:   8 */ {0x116},
	/* idx:   9 */ {0x216},
	/* idx:  10 */ {0x426},
	/* idx:  11 */ {0x826},
	/* idx:  12 */ {0x104a},
	/* idx:  13 */ {0x204a},
	/* idx:  14 */ {0x408a},
	/* idx:  15 */ {0x808a},
	/* idx:  16 */ {0x10116},
	/* idx:  17 */ {0x20116},
	/* idx:  18 */ {0x40216},
	/* idx:  19 */ {0x80216},
	/* idx:  20 */ {0x100426},
	/* idx:  21 */ {0x200426},
	/* idx:  22 */ {0x400826},
	/* idx:  23 */ {0x800826},
	/* idx:  24 */ {0x100104a},
	/* idx:  25 */ {0x200104a},
	/* idx:  26 */ {0x400204a},
	/* idx:  27 */ {0x800204a},
	/* idx:  28 */ {0x1000408a},
	/* idx:  29 */ {0x2000408a},
	/* idx:  30 */ {0x4000808a},
	/* idx:  31 */ {0x8000808a},
	/* idx:  32 */ {0x100010116},
	/* idx:  33 */ {0x200010116},
	/* idx:  34 */ {0x400020116},
	/* idx:  35 */ {0x800020116},
	/* idx:  36 */ {0x1000040216},
	/* idx:  37 */ {0x2000040216},
	/* idx:  38 */ {0x4000080216},
	/* idx:  39 */ {0x8000080216},
	/* idx:  40 */ {0x10000100426},
	/* idx:  41 */ {0x20000100426},
	/* idx:  42 */ {0x40000200426},
	/* idx:  43 */ {0x80000200426},
	/* idx:  44 */ {0x100000400826},
	/* idx:  45 */ {0x200000400826},
	/* idx:  46 */ {0x400000800826},
	/* idx:  47 */ {0x800000800826},
	/* idx:  48 */ {0x100000100104a},
	/* idx:  49 */ {0x200000100104a},
	/* idx:  50 */ {0x400000200104a},
	/* idx:  51 */ {0x800000200104a},
	/* idx:  52 */ {0x1000000400204a},
	/* idx:  53 */ {0x2000000400204a},
	/* idx:  54 */ {0x4000000800204a},
	/* idx:  55 */ {0x8000000800204a},
	/* idx:  56 */ {0x10000001000408a},
	/* idx:  57 */ {0x20000001000408a},
	/* idx:  58 */ {0x40000002000408a},
	/* idx:  59 */ {0x80000002000408a},
	/* idx:  60 */ {0x100000004000808a},
	/* idx:  61 */ {0x200000004000808a},
	/* idx:  62 */ {0x400000008000808a},
	/* idx:  63 */ {0x800000008000808a},
	/* idx:  64 */ {0x100010116, 0x1},
	/* idx:  65 */ {0x100010116, 0x2},
	/* idx:  66 */ {0x200010116, 0x4},
	/* idx:  67 */ {0x200010116, 0x8},
	/* idx:  68 */ {0x400020116, 0x10},
	/* idx:  69 */ {0x400020116, 0x20},
	/* idx:  70 */ {0x800020116, 0x40},
	/* idx:  71 */ {0x800020116, 0x80},
	/* idx:  72 */ {0x1000040216, 0x100},
	/* idx:  73 */ {0x1000040216, 0x200},
	/* idx:  74 */ {0x2000040216, 0x400},
	/* idx:  75 */ {0x2000040216, 0x800},
	/* idx:  76 */ {0x4000080216, 0x1000},
	/* idx:  77 */ {0x4000080216, 0x2000},
	/* idx:  78 */ {0x8000080216, 0x4000},
	/* idx:  79 */ {0x8000080216, 0x8000},
	/* idx:  80 */ {0x10000100426, 0x10000},
	/* idx:  81 */ {0x10000100426, 0x20000},
	/* idx:  82 */ {0x20000100426, 0x40000},
	/* idx:  83 */ {0x20000100426, 0x80000},
	/* idx:  84 */ {0x40000200426, 0x100000},
	/* idx:  85 */ {0x40000200426, 0x200000},
	/* idx:  86 */ {0x80000200426, 0x400000},
	/* idx:  87 */ {0x80000200426, 0x800000},
	/* idx:  88 */ {0x100000400826, 0x1000000},
	/* idx:  89 */ {0x100000400826, 0x2000000},
	/* idx:  90 */ {0x200000400826, 0x4000000},
	/* idx:  91 */ {0x200000400826, 0x8000000},
	/* idx:  92 */ {0x400000800826, 0x10000000},
	/* idx:  93 */ {0x400000800826, 0x20000000},
	/* idx:  94 */ {0x800000800826, 0x40000000},
	/* idx:  95 */ {0x800000800826, 0x80000000},
	/* idx:  96 */ {0x100000100104a, 0x100000000},
	/* idx:  97 */ {0x100000100104a, 0x200000000},
	/* idx:  98 */ {0x200000100104a, 0x400000000},
	/* idx:  99 */ {0x200000100104a, 0x800000000},
	/* idx: 100 */ {0x400000200104a, 0x1000000000},
	/* idx: 101 */ {0x400000200104a, 0x2000000000},
	/* idx: 102 */ {0x800000200104a, 0x4000000000},
	/* idx: 103 */ {0x800000200104a, 0x8000000000},
	/* idx: 104 */ {0x1000000400204a, 0x10000000000},
	/* idx: 105 */ {0x1000000400204a, 0x20000000000},
	/* idx: 106 */ {0x2000000400204a, 0x40000000000},
	/* idx: 107 */ {0x2000000400204a, 0x80000000000},
	/* idx: 108 */ {0x4000000800204a, 0x100000000000},
	/* idx: 109 */ {0x4000000800204a, 0x200000000000},
	/* idx: 110 */ {0x8000000800204a, 0x400000000000},
	/* idx: 111 */ {0x8000000800204a, 0x800000000000},
	/* idx: 112 */ {0x10000001000408a, 0x1000000000000},
	/* idx: 113 */ {0x10000001000408a, 0x2000000000000},
	/* idx: 114 */ {0x20000001000408a, 0x4000000000000},
	/* idx: 115 */ {0x20000001000408a, 0x8000000000000},
	/* idx: 116 */ {0x40000002000408a, 0x10000000000000},
	/* idx: 117 */ {0x40000002000408a, 0x20000000000000},
	/* idx: 118 */ {0x80000002000408a, 0x40000000000000},
	/* idx: 119 */ {0x80000002000408a, 0x80000000000000},
	/* idx: 120 */ {0x100000004000808a, 0x100000000000000},
	/* idx: 121 */ {0x100000004000808a, 0x200000000000000},
	/* idx: 122 */ {0x200000004000808a, 0x400000000000000},
	/* idx: 123 */ {0x200000004000808a, 0x800000000000000},
	/* idx: 124 */ {0x400000008000808a, 0x1000000000000000},
	/* idx: 125 */ {0x400000008000808a, 0x2000000000000000},
	/* idx: 126 */ {0x800000008000808a, 0x4000000000000000},
	/* idx: 127 */ {0x800000008000808a, 0x8000000000000000},
	/* idx: 128 */ {0x100010116, 0x1, 0x1},
	/* idx: 129 */ {0x100010116, 0x1, 0x2},
	/* idx: 130 */ {0x100010116, 0x2, 0x4},
	/* idx: 131 */ {0x100010116, 0x2, 0x8},
	/* idx: 132 */ {0x200010116, 0x4, 0x10},
	/* idx: 133 */ {0x200010116, 0x4, 0x20},
	/* idx: 134 */ {0x200010116, 0x8, 0x40},
	/* idx: 135 */ {0x200010116, 0x8, 0x80},
	/* idx: 136 */ {0x400020116, 0x10, 0x100},
	/* idx: 137 */ {0x400020116, 0x10, 0x200},
	/* idx: 138 */ {0x400020116, 0x20, 0x400},
	/* idx: 139 */ {0x400020116, 0x20, 0x800},
	/* idx: 140 */ {0x800020116, 0x40, 0x1000},
	/* idx: 141 */ {0x800020116, 0x40, 0x2000},
	/* idx: 142 */ {0x800020116, 0x80, 0x4000},
	/* idx: 143 */ {0x800020116, 0x80, 0x8000},
	/* idx: 144 */ {0x1000040216, 0x100, 0x10000},
	/* idx: 145 */ {0x1000040216, 0x100, 0x20000},
	/* idx: 146 */ {0x1000040216, 0x200, 0x40000},
	/* idx: 147 */ {0x1000040216, 0x200, 0x80000},
	/* idx: 148 */ {0x2000040216, 0x400, 0x100000},
	/* idx: 149 */ {0x2000040216, 0x400, 0x200000},
	/* idx: 150 */ {0x2000040216, 0x800, 0x400000},
	/* idx: 151 */ {0x2000040216, 0x800, 0x800000},
	/* idx: 152 */ {0x4000080216, 0x1000, 0x1000000},
	/* idx: 153 */ {0x4000080216, 0x1000, 0x2000000},
	/* idx: 154 */ {0x4000080216, 0x2000, 0x4000000},
	/* idx: 155 */ {0x4000080216, 0x2000, 0x8000000},
	/* idx: 156 */ {0x8000080216, 0x4000, 0x10000000},
	/* idx: 157 */ {0x8000080216, 0x4000, 0x20000000},
	/* idx: 158 */ {0x8000080216, 0x8000, 0x40000000},
	/* idx: 159 */ {0x8000080216, 0x8000, 0x80000000},
	/* idx: 160 */ {0x10000100426, 0x10000, 0x100000000},
	/* idx: 161 */ {0x10000100426, 0x10000, 0x200000000},
	/* idx: 162 */ {0x10000100426, 0x20000, 0x400000000},
	/* idx: 163 */ {0x10000100426, 0x20000, 0x800000000},
	/* idx: 164 */ {0x20000100426, 0x40000, 0x1000000000},
	/* idx: 165 */ {0x20000100426, 0x40000, 0x2000000000},
	/* idx: 166 */ {0x20000100426, 0x80000, 0x4000000000},
	/* idx: 167 */ {0x20000100426, 0x80000, 0x8000000000},
	/* idx: 168 */ {0x40000200426, 0x100000, 0x10000000000},
	/* idx: 169 */ {0x40000200426, 0x100000, 0x20000000000},
	/* idx: 170 */ {0x40000200426, 0x200000, 0x40000000000},
	/* idx: 171 */ {0x40000200426, 0x200000, 0x80000000000},
	/* idx: 172 */ {0x80000200426, 0x400000, 0x100000000000},
	/* idx: 173 */ {0x80000200426, 0x400000, 0x200000000000},
	/* idx: 174 */ {0x80000200426, 0x800000, 0x400000000000},
	/* idx: 175 */ {0x80000200426, 0x800000, 0x800000000000},
	/* idx: 176 */ {0x100000400826, 0x1000000, 0x1000000000000},
	/* idx: 177 */ {0x100000400826, 0x1000000, 0x2000000000000},
	/* idx: 178 */ {0x100000400826, 0x2000000, 0x4000000000000},
	/* idx: 179 */ {0x100000400826, 0x2000000, 0x8000000000000},
	/* idx: 180 */ {0x200000400826, 0x4000000, 0x10000000000000},
	/* idx: 181 */ {0x200000400826, 0x4000000, 0x20000000000000},
	/* idx: 182 */ {0x200000400826, 0x8000000, 0x40000000000000},
	/* idx: 183 */ {0x200000400826, 0x8000000, 0x80000000000000},
	/* idx: 184 */ {0x400000800826, 0x10000000, 0x100000000000000},
	/* idx: 185 */ {0x400000800826, 0x10000000, 0x200000000000000},
	/* idx: 186 */ {0x400000800826, 0x20000000, 0x400000000000000},
	/* idx: 187 */ {0x400000800826, 0x20000000, 0x800000000000000},
	/* idx: 188 */ {0x800000800826, 0x40000000, 0x1000000000000000},
	/* idx: 189 */ {0x800000800826, 0x40000000, 0x2000000000000000},
	/* idx: 190 */ {0x800000800826, 0x80000000, 0x4000000000000000},
	/* idx: 191 */ {0x800000800826, 0x80000000, 0x8000000000000000},
	/* idx: 192 */ {0x100000100104a, 0x100000000, 0x0, 0x1},
	/* idx: 193 */ {0x100000100104a, 0x100000000, 0x0, 0x2},
	/* idx: 194 */ {0x100000100104a, 0x200000000, 0x0, 0x4},
	/* idx: 195 */ {0x100000100104a, 0x200000000, 0x0, 0x8},
	/* idx: 196 */ {0x200000100104a, 0x400000000, 0x0, 0x10},
	/* idx: 197 */ {0x200000100104a, 0x400000000, 0x0, 0x20},
	/* idx: 198 */ {0x200000100104a, 0x800000000, 0x0, 0x40},
	/* idx: 199 */ {0x200000100104a, 0x800000000, 0x0, 0x80},
	/* idx: 200 */ {0x400000200104a, 0x1000000000, 0x0, 0x100},
	/* idx: 201 */ {0x400000200104a, 0x1000000000, 0x0, 0x200},
	/* idx: 202 */ {0x400000200104a, 0x2000000000, 0x0, 0x400},
	/* idx: 203 */ {0x400000200104a, 0x2000000000, 0x0, 0x800},
	/* idx: 204 */ {0x800000200104a, 0x4000000000, 0x0, 0x1000},
	/* idx: 205 */ {0x800000200104a, 0x4000000000, 0x0, 0x2000},
	/* idx: 206 */ {0x800000200104a, 0x8000000000, 0x0, 0x4000},
	/* idx: 207 */ {0x800000200104a, 0x8000000000, 0x0, 0x8000},
	/* idx: 208 */ {0x1000000400204a, 0x10000000000, 0x0, 0x10000},
	/* idx: 209 */ {0x1000000400204a, 0x10000000000, 0x0, 0x20000},
	/* idx: 210 */ {0x1000000400204a, 0x20000000000, 0x0, 0x40000},
	/* idx: 211 */ {0x1000000400204a, 0x20000000000, 0x0, 0x80000},
	/* idx: 212 */ {0x2000000400204a, 0x40000000000, 0x0, 0x100000},
	/* idx: 213 */ {0x2000000400204a, 0x40000000000, 0x0, 0x200000},
	/* idx: 214 */ {0x2000000400204a, 0x80000000000, 0x0, 0x400000},
	/* idx: 215 */ {0x2000000400204a, 0x80000000000, 0x0, 0x800000},
	/* idx: 216 */ {0x4000000800204a, 0x100000000000, 0x0, 0x1000000},
	/* idx: 217 */ {0x4000000800204a, 0x100000000000, 0x0, 0x2000000},
	/* idx: 218 */ {0x4000000800204a, 0x200000000000, 0x0, 0x4000000},
	/* idx: 219 */ {0x4000000800204a, 0x200000000000, 0x0, 0x8000000},
	/* idx: 220 */ {0x8000000800204a, 0x400000000000, 0x0, 0x10000000},
	/* idx: 221 */ {0x8000000800204a, 0x400000000000, 0x0, 0x20000000},
	/* idx: 222 */ {0x8000000800204a, 0x800000000000, 0x0, 0x40000000},
	/* idx: 223 */ {0x8000000800204a, 0x800000000000, 0x0, 0x80000000},
	/* idx: 224 */ {0x10000001000408a, 0x1000000000000, 0x0, 0x100000000},
	/* idx: 225 */ {0x10000001000408a, 0x1000000000000, 0x0, 0x200000000},
	/* idx: 226 */ {0x10000001000408a, 0x2000000000000, 0x0, 0x400000000},
	/* idx: 227 */ {0x10000001000408a, 0x2000000000000, 0x0, 0x800000000},
	/* idx: 228 */ {0x20000001000408a, 0x4000000000000, 0x0, 0x1000000000},
	/* idx: 229 */ {0x20000001000408a, 0x4000000000000, 0x0, 0x2000000000},
	/* idx: 230 */ {0x20000001000408a, 0x8000000000000, 0x0, 0x4000000000},
	/* idx: 231 */ {0x20000001000408a, 0x8000000000000, 0x0, 0x8000000000},
	/* idx: 232 */ {0x40000002000408a, 0x10000000000000, 0x0, 0x10000000000},
	/* idx: 233 */ {0x40000002000408a, 0x10000000000000, 0x0, 0x20000000000},
	/* idx: 234 */ {0x40000002000408a, 0x20000000000000, 0x0, 0x40000000000},
	/* idx: 235 */ {0x40000002000408a, 0x20000000000000, 0x0, 0x80000000000},
	/* idx: 236 */ {0x80000002000408a, 0x40000000000000, 0x0, 0x100000000000},
	/* idx: 237 */ {0x80000002000408a, 0x40000000000000, 0x0, 0x200000000000},
	/* idx: 238 */ {0x80000002000408a, 0x80000000000000, 0x0, 0x400000000000},
	/* idx: 239 */ {0x80000002000408a, 0x80000000000000, 0x0, 0x800000000000},
	/* idx: 240 */ {0x100000004000808a, 0x100000000000000, 0x0, 0x1000000000000},
	/* idx: 241 */ {0x100000004000808a, 0x100000000000000, 0x0, 0x2000000000000},
	/* idx: 242 */ {0x100000004000808a, 0x200000000000000, 0x0, 0x4000000000000},
	/* idx: 243 */ {0x100000004000808a, 0x200000000000000, 0x0, 0x8000000000000},
	/* idx: 244 */ {0x200000004000808a, 0x400000000000000, 0x0, 0x10000000000000},
	/* idx: 245 */ {0x200000004000808a, 0x400000000000000, 0x0, 0x20000000000000},
	/* idx: 246 */ {0x200000004000808a, 0x800000000000000, 0x0, 0x40000000000000},
	/* idx: 247 */ {0x200000004000808a, 0x800000000000000, 0x0, 0x80000000000000},
	/* idx: 248 */ {0x400000008000808a, 0x1000000000000000, 0x0, 0x100000000000000},
	/* idx: 249 */ {0x400000008000808a, 0x1000000000000000, 0x0, 0x200000000000000},
	/* idx: 250 */ {0x400000008000808a, 0x2000000000000000, 0x0, 0x400000000000000},
	/* idx: 251 */ {0x400000008000808a, 0x2000000000000000, 0x0, 0x800000000000000},
	/* idx: 252 */ {0x800000008000808a, 0x4000000000000000, 0x0, 0x1000000000000000},
	/* idx: 253 */ {0x800000008000808a, 0x4000000000000000, 0x0, 0x2000000000000000},
	/* idx: 254 */ {0x800000008000808a, 0x8000000000000000, 0x0, 0x4000000000000000},
	/* idx: 255 */ {0x800000008000808a, 0x8000000000000000, 0x0, 0x8000000000000000},
	/* idx: 256 */ {0x100010116, 0x1, 0x1, 0x0, 0x1},
	/* idx: 257 */ {0x100010116, 0x1, 0x1, 0x0, 0x2},
	/* idx: 258 */ {0x100010116, 0x1, 0x2, 0x0, 0x4},
	/* idx: 259 */ {0x100010116, 0x1, 0x2, 0x0, 0x8},
	/* idx: 260 */ {0x100010116, 0x2, 0x4, 0x0, 0x10},
	/* idx: 261 */ {0x100010116, 0x2, 0x4, 0x0, 0x20},
	/* idx: 262 */ {0x100010116, 0x2, 0x8, 0x0, 0x40},
	/* idx: 263 */ {0x100010116, 0x2, 0x8, 0x0, 0x80},
	/* idx: 264 */ {0x200010116, 0x4, 0x10, 0x0, 0x100},
	/* idx: 265 */ {0x200010116, 0x4, 0x10, 0x0, 0x200},
	/* idx: 266 */ {0x200010116, 0x4, 0x20, 0x0, 0x400},
	/* idx: 267 */ {0x200010116, 0x4, 0x20, 0x0, 0x800},
	/* idx: 268 */ {0x200010116, 0x8, 0x40, 0x0, 0x1000},
	/* idx: 269 */ {0x200010116, 0x8, 0x40, 0x0, 0x2000},
	/* idx: 270 */ {0x200010116, 0x8, 0x80, 0x0, 0x4000},
	/* idx: 271 */ {0x200010116, 0x8, 0x80, 0x0, 0x8000},
	/* idx: 272 */ {0x400020116, 0x10, 0x100, 0x0, 0x10000},
	/* idx: 273 */ {0x400020116, 0x10, 0x100, 0x0, 0x20000},
	/* idx: 274 */ {0x400020116, 0x10, 0x200, 0x0, 0x40000},
	/* idx: 275 */ {0x400020116, 0x10, 0x200, 0x0, 0x80000},
	/* idx: 276 */ {0x400020116, 0x20, 0x400, 0x0, 0x100000},
	/* idx: 277 */ {0x400020116, 0x20, 0x400, 0x0, 0x200000},
	/* idx: 278 */ {0x400020116, 0x20, 0x800, 0x0, 0x400000},
	/* idx: 279 */ {0x400020116, 0x20, 0x800, 0x0, 0x800000},
	/* idx: 280 */ {0x800020116, 0x40, 0x1000, 0x0, 0x1000000},
	/* idx: 281 */ {0x800020116, 0x40, 0x1000, 0x0, 0x2000000},
	/* idx: 282 */ {0x800020116, 0x40, 0x2000, 0x0, 0x4000000},
	/* idx: 283 */ {0x800020116, 0x40, 0x2000, 0x0, 0x8000000},
	/* idx: 284 */ {0x800020116, 0x80, 0x4000, 0x0, 0x10000000},
	/* idx: 285 */ {0x800020116, 0x80, 0x4000, 0x0, 0x20000000},
	/* idx: 286 */ {0x800020116, 0x80, 0x8000, 0x0, 0x40000000},
	/* idx: 287 */ {0x800020116, 0x80, 0x8000, 0x0, 0x80000000},
	/* idx: 288 */ {0x1000040216, 0x100, 0x10000, 0x0, 0x100000000},
	/* idx: 289 */ {0x1000040216, 0x100, 0x10000, 0x0, 0x200000000},
	/* idx: 290 */ {0x1000040216, 0x100, 0x20000, 0x0, 0x400000000},
	/* idx: 291 */ {0x1000040216, 0x100, 0x20000, 0x0, 0x800000000},
	/* idx: 292 */ {0x1000040216, 0x200, 0x40000, 0x0, 0x1000000000},
	/* idx: 293 */ {0x1000040216, 0x200, 0x40000, 0x0, 0x2000000000},
	/* idx: 294 */ {0x1000040216, 0x200, 0x80000, 0x0, 0x4000000000},
	/* idx: 295 */ {0x1000040216, 0x200, 0x80000, 0x0, 0x8000000000},
	/* idx: 296 */ {0x2000040216, 0x400, 0x100000, 0x0, 0x10000000000},
	/* idx: 297 */ {0x2000040216, 0x400, 0x100000, 0x0, 0x20000000000},
	/* idx: 298 */ {0x2000040216, 0x400, 0x200000, 0x0, 0x40000000000},
	/* idx: 299 */ {0x2000040216, 0x400, 0x200000, 0x0, 0x80000000000},
	/* idx: 300 */ {0x2000040216, 0x800, 0x400000, 0x0, 0x100000000000},
	/* idx: 301 */ {0x2000040216, 0x800, 0x400000, 0x0, 0x200000000000},
	/* idx: 302 */ {0x2000040216, 0x800, 0x800000, 0x0, 0x400000000000},
	/* idx: 303 */ {0x2000040216, 0x800, 0x800000, 0x0, 0x800000000000},
	/* idx: 304 */ {0x4000080216, 0x1000, 0x1000000, 0x0, 0x1000000000000},
	/* idx: 305 */ {0x4000080216, 0x1000, 0x1000000, 0x0, 0x2000000000000},
	/* idx: 306 */ {0x4000080216, 0x1000, 0x2000000, 0x0, 0x4000000000000},
	/* idx: 307 */ {0x4000080216, 0x1000, 0x2000000, 0x0, 0x8000000000000},
	/* idx: 308 */ {0x4000080216, 0x2000, 0x4000000, 0x0, 0x10000000000000},
	/* idx: 309 */ {0x4000080216, 0x2000, 0x4000000, 0x0, 0x20000000000000},
	/* idx: 310 */ {0x4000080216, 0x2000, 0x8000000, 0x0, 0x40000000000000},
	/* idx: 311 */ {0x4000080216, 0x2000, 0x8000000, 0x0, 0x80000000000000},
	/* idx: 312 */ {0x8000080216, 0x4000, 0x10000000, 0x0, 0x100000000000000},
	/* idx: 313 */ {0x8000080216, 0x4000, 0x10000000, 0x0, 0x200000000000000},
	/* idx: 314 */ {0x8000080216, 0x4000, 0x20000000, 0x0, 0x400000000000000},
	/* idx: 315 */ {0x8000080216, 0x4000, 0x20000000, 0x0, 0x800000000000000},
	/* idx: 316 */ {0x8000080216, 0x8000, 0x40000000, 0x0, 0x1000000000000000},
	/* idx: 317 */ {0x8000080216, 0x8000, 0x40000000, 0x0, 0x2000000000000000},
	/* idx: 318 */ {0x8000080216, 0x8000, 0x80000000, 0x0, 0x4000000000000000},
	/* idx: 319 */ {0x8000080216, 0x8000, 0x80000000, 0x0, 0x8000000000000000},
	/* idx: 320 */ {0x10000100426, 0x10000, 0x100000000, 0x0, 0x0, 0x1},
	/* idx: 321 */ {0x10000100426, 0x10000, 0x100000000, 0x0, 0x0, 0x2},
	/* idx: 322 */ {0x10000100426, 0x10000, 0x200000000, 0x0, 0x0, 0x4},
	/* idx: 323 */ {0x10000100426, 0x10000, 0x200000000, 0x0, 0x0, 0x8},
	/* idx: 324 */ {0x10000100426, 0x20000, 0x400000000, 0x0, 0x0, 0x10},
	/* idx: 325 */ {0x10000100426, 0x20000, 0x400000000, 0x0, 0x0, 0x20},
	/* idx: 326 */ {0x10000100426, 0x20000, 0x800000000, 0x0, 0x0, 0x40},
	/* idx: 327 */ {0x10000100426, 0x20000, 0x800000000, 0x0, 0x0, 0x80},
	/* idx: 328 */ {0x20000100426, 0x40000, 0x1000000000, 0x0, 0x0, 0x100},
	/* idx: 329 */ {0x20000100426, 0x40000, 0x1000000000, 0x0, 0x0, 0x200},
	/* idx: 330 */ {0x20000100426, 0x40000, 0x2000000000, 0x0, 0x0, 0x400},
	/* idx: 331 */ {0x20000100426, 0x40000, 0x2000000000, 0x0, 0x0, 0x800},
	/* idx: 332 */ {0x20000100426, 0x80000, 0x4000000000, 0x0, 0x0, 0x1000},
	/* idx: 333 */ {0x20000100426, 0x80000, 0x4000000000, 0x0, 0x0, 0x2000},
	/* idx: 334 */ {0x20000100426, 0x80000, 0x8000000000, 0x0, 0x0, 0x4000},
	/* idx: 335 */ {0x20000100426, 0x80000, 0x8000000000, 0x0, 0x0, 0x8000},
	/* idx: 336 */ {0x40000200426, 0x100000, 0x10000000000, 0x0, 0x0, 0x10000},
	/* idx: 337 */ {0x40000200426, 0x100000, 0x10000000000, 0x0, 0x0, 0x20000},
	/* idx: 338 */ {0x40000200426, 0x100000, 0x20000000000, 0x0, 0x0, 0x40000},
	/* idx: 339 */ {0x40000200426, 0x100000, 0x20000000000, 0x0, 0x0, 0x80000},
	/* idx: 340 */ {0x40000200426, 0x200000, 0x40000000000, 0x0, 0x0, 0x100000},
	/* idx: 341 */ {0x40000200426, 0x200000, 0x40000000000, 0x0, 0x0, 0x200000},
	/* idx: 342 */ {0x40000200426, 0x200000, 0x80000000000, 0x0, 0x0, 0x400000},
	/* idx: 343 */ {0x40000200426, 0x200000, 0x80000000000, 0x0, 0x0, 0x800000},
	/* idx: 344 */ {0x80000200426, 0x400000, 0x100000000000, 0x0, 0x0, 0x1000000},
	/* idx: 345 */ {0x80000200426, 0x400000, 0x100000000000, 0x0, 0x0, 0x2000000},
	/* idx: 346 */ {0x80000200426, 0x400000, 0x200000000000, 0x0, 0x0, 0x4000000},
	/* idx: 347 */ {0x80000200426, 0x400000, 0x200000000000, 0x0, 0x0, 0x8000000},
	/* idx: 348 */ {0x80000200426, 0x800000, 0x400000000000, 0x0, 0x0, 0x10000000},
	/* idx: 349 */ {0x80000200426, 0x800000, 0x400000000000, 0x0, 0x0, 0x20000000},
	/* idx: 350 */ {0x80000200426, 0x800000, 0x800000000000, 0x0, 0x0, 0x40000000},
	/* idx: 351 */ {0x80000200426, 0x800000, 0x800000000000, 0x0, 0x0, 0x80000000},
	/* idx: 352 */ {0x100000400826, 0x1000000, 0x1000000000000, 0x0, 0x0, 0x100000000},
	/* idx: 353 */ {0x100000400826, 0x1000000, 0x1000000000000, 0x0, 0x0, 0x200000000},
	/* idx: 354 */ {0x100000400826, 0x1000000, 0x2000000000000, 0x0, 0x0, 0x400000000},
	/* idx: 355 */ {0x100000400826, 0x1000000, 0x2000000000000, 0x0, 0x0, 0x800000000},
	/* idx: 356 */ {0x100000400826, 0x2000000, 0x4000000000000, 0x0, 0x0, 0x1000000000},
	/* idx: 357 */ {0x100000400826, 0x2000000, 0x4000000000000, 0x0, 0x0, 0x2000000000},
	/* idx: 358 */ {0x100000400826, 0x2000000, 0x8000000000000, 0x0, 0x0, 0x4000000000},
	/* idx: 359 */ {0x100000400826, 0x2000000, 0x8000000000000, 0x0, 0x0, 0x8000000000},
	/* idx: 360 */ {0x200000400826, 0x4000000, 0x10000000000000, 0x0, 0x0, 0x10000000000},
	/* idx: 361 */ {0x200000400826, 0x4000000, 0x10000000000000, 0x0, 0x0, 0x20000000000},
	/* idx: 362 */ {0x200000400826, 0x4000000, 0x20000000000000, 0x0, 0x0, 0x40000000000},
	/* idx: 363 */ {0x200000400826, 0x4000000, 0x20000000000000, 0x0, 0x0, 0x80000000000},
	/* idx: 364 */ {0x200000400826, 0x8000000, 0x40000000000000, 0x0, 0x0, 0x100000000000},
	/* idx: 365 */ {0x200000400826, 0x8000000, 0x40000000000000, 0x0, 0x0, 0x200000000000},
	/* idx: 366 */ {0x200000400826, 0x8000000, 0x80000000000000, 0x0, 0x0, 0x400000000000},
	/* idx: 367 */ {0x200000400826, 0x8000000, 0x80000000000000, 0x0, 0x0, 0x800000000000},
	/* idx: 368 */ {0x400000800826, 0x10000000, 0x100000000000000, 0x0, 0x0, 0x1000000000000},
	/* idx: 369 */ {0x400000800826, 0x10000000, 0x100000000000000, 0x0, 0x0, 0x2000000000000},
	/* idx: 370 */ {0x400000800826, 0x10000000, 0x200000000000000, 0x0, 0x0, 0x4000000000000},
	/* idx: 371 */ {0x400000800826, 0x10000000, 0x200000000000000, 0x0, 0x0, 0x8000000000000},
	/* idx: 372 */ {0x400000800826, 0x20000000, 0x400000000000000, 0x0, 0x0, 0x10000000000000},
	/* idx: 373 */ {0x400000800826, 0x20000000, 0x400000000000000, 0x0, 0x0, 0x20000000000000},
	/* idx: 374 */ {0x400000800826, 0x20000000, 0x800000000000000, 0x0, 0x0, 0x40000000000000},
	/* idx: 375 */ {0x400000800826, 0x20000000, 0x800000000000000, 0x0, 0x0, 0x80000000000000},
	/* idx: 376 */ {0x800000800826, 0x40000000, 0x1000000000000000, 0x0, 0x0, 0x100000000000000},
	/* idx: 377 */ {0x800000800826, 0x40000000, 0x1000000000000000, 0x0, 0x0, 0x200000000000000},
	/* idx: 378 */ {0x800000800826, 0x40000000, 0x2000000000000000, 0x0, 0x0, 0x400000000000000},
	/* idx: 379 */ {0x800000800826, 0x40000000, 0x2000000000000000, 0x0, 0x0, 0x800000000000000},
	/* idx: 380 */ {0x800000800826, 0x80000000, 0x4000000000000000, 0x0, 0x0, 0x1000000000000000},
	/* idx: 381 */ {0x800000800826, 0x80000000, 0x4000000000000000, 0x0, 0x0, 0x2000000000000000},
	/* idx: 382 */ {0x800000800826, 0x80000000, 0x8000000000000000, 0x0, 0x0, 0x4000000000000000},
	/* idx: 383 */ {0x800000800826, 0x80000000, 0x8000000000000000, 0x0, 0x0, 0x8000000000000000},
	/* idx: 384 */ {0x100000100104a, 0x100000000, 0x0, 0x1, 0x0, 0x0, 0x1},
	/* idx: 385 */ {0x100000100104a, 0x100000000, 0x0, 0x1, 0x0, 0x0, 0x2},
	/* idx: 386 */ {0x100000100104a, 0x100000000, 0x0, 0x2, 0x0, 0x0, 0x4},
	/* idx: 387 */ {0x100000100104a, 0x100000000, 0x0, 0x2, 0x0, 0x0, 0x8},
	/* idx: 388 */ {0x100000100104a, 0x200000000, 0x0, 0x4, 0x0, 0x0, 0x10},
	/* idx: 389 */ {0x100000100104a, 0x200000000, 0x0, 0x4, 0x0, 0x0, 0x20},
	/* idx: 390 */ {0x100000100104a, 0x200000000, 0x0, 0x8, 0x0, 0x0, 0x40},
	/* idx: 391 */ {0x100000100104a, 0x200000000, 0x0, 0x8, 0x0, 0x0, 0x80},
	/* idx: 392 */ {0x200000100104a, 0x400000000, 0x0, 0x10, 0x0, 0x0, 0x100},
	/* idx: 393 */ {0x200000100104a, 0x400000000, 0x0, 0x10, 0x0, 0x0, 0x200},
	/* idx: 394 */ {0x200000100104a, 0x400000000, 0x0, 0x20, 0x0, 0x0, 0x400},
	/* idx: 395 */ {0x200000100104a, 0x400000000, 0x0, 0x20, 0x0, 0x0, 0x800},
	/* idx: 396 */ {0x200000100104a, 0x800000000, 0x0, 0x40, 0x0, 0x0, 0x1000},
	/* idx: 397 */ {0x200000100104a, 0x800000000, 0x0, 0x40, 0x0, 0x0, 0x2000},
	/* idx: 398 */ {0x200000100104a, 0x800000000, 0x0, 0x80, 0x0, 0x0, 0x4000},
	/* idx: 399 */ {0x200000100104a, 0x800000000, 0x0, 0x80, 0x0, 0x0, 0x8000},
	/* idx: 400 */ {0x400000200104a, 0x1000000000, 0x0, 0x100, 0x0, 0x0, 0x10000},
	/* idx: 401 */ {0x400000200104a, 0x1000000000, 0x0, 0x100, 0x0, 0x0, 0x20000},
	/* idx: 402 */ {0x400000200104a, 0x1000000000, 0x0, 0x200, 0x0, 0x0, 0x40000},
	/* idx: 403 */ {0x400000200104a, 0x1000000000, 0x0, 0x200, 0x0, 0x0, 0x80000},
	/* idx: 404 */ {0x400000200104a, 0x2000000000, 0x0, 0x400, 0x0, 0x0, 0x100000},
	/* idx: 405 */ {0x400000200104a, 0x2000000000, 0x0, 0x400, 0x0, 0x0, 0x200000},
	/* idx: 406 */ {0x400000200104a, 0x2000000000, 0x0, 0x800, 0x0, 0x0, 0x400000},
	/* idx: 407 */ {0x400000200104a, 0x2000000000, 0x0, 0x800, 0x0, 0x0, 0x800000},
	/* idx: 408 */ {0x800000200104a, 0x4000000000, 0x0, 0x1000, 0x0, 0x0, 0x1000000},
	/* idx: 409 */ {0x800000200104a, 0x4000000000, 0x0, 0x1000, 0x0, 0x0, 0x2000000},
	/* idx: 410 */ {0x800000200104a, 0x4000000000, 0x0, 0x2000, 0x0, 0x0, 0x4000000},
	/* idx: 411 */ {0x800000200104a, 0x4000000000, 0x0, 0x2000, 0x0, 0x0, 0x8000000},
	/* idx: 412 */ {0x800000200104a, 0x8000000000, 0x0, 0x4000, 0x0, 0x0, 0x10000000},
	/* idx: 413 */ {0x800000200104a, 0x8000000000, 0x0, 0x4000, 0x0, 0x0, 0x20000000},
	/* idx: 414 */ {0x800000200104a, 0x8000000000, 0x0, 0x8000, 0x0, 0x0, 0x40000000},
	/* idx: 415 */ {0x800000200104a, 0x8000000000, 0x0, 0x8000, 0x0, 0x0, 0x80000000},
	/* idx: 416 */ {0x1000000400204a, 0x10000000000, 0x0, 0x10000, 0x0, 0x0, 0x100000000},
	/* idx: 417 */ {0x1000000400204a, 0x10000000000, 0x0, 0x10000, 0x0, 0x0, 0x200000000},
	/* idx: 418 */ {0x1000000400204a, 0x10000000000, 0x0, 0x20000, 0x0, 0x0, 0x400000000},
	/* idx: 419 */ {0x1000000400204a, 0x10000000000, 0x0, 0x20000, 0x0, 0x0, 0x800000000},
	/* idx: 420 */ {0x1000000400204a, 0x20000000000, 0x0, 0x40000, 0x0, 0x0, 0x1000000000},
	/* idx: 421 */ {0x1000000400204a, 0x20000000000, 0x0, 0x40000, 0x0, 0x0, 0x2000000000},
	/* idx: 422 */ {0x1000000400204a, 0x20000000000, 0x0, 0x80000, 0x0, 0x0, 0x4000000000},
	/* idx: 423 */ {0x1000000400204a, 0x20000000000, 0x0, 0x80000, 0x0, 0x0, 0x8000000000},
	/* idx: 424 */ {0x2000000400204a, 0x40000000000, 0x0, 0x100000, 0x0, 0x0, 0x10000000000},
	/* idx: 425 */ {0x2000000400204a, 0x40000000000, 0x0, 0x100000, 0x0, 0x0, 0x20000000000},
	/* idx: 426 */ {0x2000000400204a, 0x40000000000, 0x0, 0x200000, 0x0, 0x0, 0x40000000000},
	/* idx: 427 */ {0x2000000400204a, 0x40000000000, 0x0, 0x200000, 0x0, 0x0, 0x80000000000},
	/* idx: 428 */ {0x2000000400204a, 0x80000000000, 0x0, 0x400000, 0x0, 0x0, 0x100000000000},
	/* idx: 429 */ {0x2000000400204a, 0x80000000000, 0x0, 0x400000, 0x0, 0x0, 0x200000000000},
	/* idx: 430 */ {0x2000000400204a, 0x80000000000, 0x0, 0x800000, 0x0, 0x0, 0x400000000000},
	/* idx: 431 */ {0x2000000400204a, 0x80000000000, 0x0, 0x800000, 0x0, 0x0, 0x800000000000},
	/* idx: 432 */ {0x4000000800204a, 0x100000000000, 0x0, 0x1000000, 0x0, 0x0, 0x1000000000000},
	/* idx: 433 */ {0x4000000800204a, 0x100000000000, 0x0, 0x1000000, 0x0, 0x0, 0x2000000000000},
	/* idx: 434 */ {0x4000000800204a, 0x100000000000, 0x0, 0x2000000, 0x0, 0x0, 0x4000000000000},
	/* idx: 435 */ {0x4000000800204a, 0x100000000000, 0x0, 0x2000000, 0x0, 0x0, 0x8000000000000},
	/* idx: 436 */ {0x4000000800204a, 0x200000000000, 0x0, 0x4000000, 0x0, 0x0, 0x10000000000000},
	/* idx: 437 */ {0x4000000800204a, 0x200000000000, 0x0, 0x4000000, 0x0, 0x0, 0x20000000000000},
	/* idx: 438 */ {0x4000000800204a, 0x200000000000, 0x0, 0x8000000, 0x0, 0x0, 0x40000000000000},
	/* idx: 439 */ {0x4000000800204a, 0x200000000000, 0x0, 0x8000000, 0x0, 0x0, 0x80000000000000},
	/* idx: 440 */ {0x8000000800204a, 0x400000000000, 0x0, 0x10000000, 0x0, 0x0, 0x100000000000000},
	/* idx: 441 */ {0x8000000800204a, 0x400000000000, 0x0, 0x10000000, 0x0, 0x0, 0x200000000000000},
	/* idx: 442 */ {0x8000000800204a, 0x400000000000, 0x0, 0x20000000, 0x0, 0x0, 0x400000000000000},
	/* idx: 443 */ {0x8000000800204a, 0x400000000000, 0x0, 0x20000000, 0x0, 0x0, 0x800000000000000},
	/* idx: 444 */ {0x8000000800204a, 0x800000000000, 0x0, 0x40000000, 0x0, 0x0, 0x1000000000000000},
	/* idx: 445 */ {0x8000000800204a, 0x800000000000, 0x0, 0x40000000, 0x0, 0x0, 0x2000000000000000},
	/* idx: 446 */ {0x8000000800204a, 0x800000000000, 0x0, 0x80000000, 0x0, 0x0, 0x4000000000000000},
	/* idx: 447 */ {0x8000000800204a, 0x800000000000, 0x0, 0x80000000, 0x0, 0x0, 0x8000000000000000},
	/* idx: 448 */ {0x10000001000408a, 0x1000000000000, 0x0, 0x100000000, 0x0, 0x0, 0x0, 0x1},
	/* idx: 449 */ {0x10000001000408a, 0x1000000000000, 0x0, 0x100000000, 0x0, 0x0, 0x0, 0x2},
	/* idx: 450 */ {0x10000001000408a, 0x1000000000000, 0x0, 0x200000000, 0x0, 0x0, 0x0, 0x4},
	/* idx: 451 */ {0x10000001000408a, 0x1000000000000, 0x0, 0x200000000, 0x0, 0x0, 0x0, 0x8},
	/* idx: 452 */ {0x10000001000408a, 0x2000000000000, 0x0, 0x400000000, 0x0, 0x0, 0x0, 0x10},
	/* idx: 453 */ {0x10000001000408a, 0x2000000000000, 0x0, 0x400000000, 0x0, 0x0, 0x0, 0x20},
	/* idx: 454 */ {0x10000001000408a, 0x2000000000000, 0x0, 0x800000000, 0x0, 0x0, 0x0, 0x40},
	/* idx: 455 */ {0x10000001000408a, 0x2000000000000, 0x0, 0x800000000, 0x0, 0x0, 0x0, 0x80},
	/* idx: 456 */ {0x20000001000408a, 0x4000000000000, 0x0, 0x1000000000, 0x0, 0x0, 0x0, 0x100},
	/* idx: 457 */ {0x20000001000408a, 0x4000000000000, 0x0, 0x1000000000, 0x0, 0x0, 0x0, 0x200},
	/* idx: 458 */ {0x20000001000408a, 0x4000000000000, 0x0, 0x2000000000, 0x0, 0x0, 0x0, 0x400},
	/* idx: 459 */ {0x20000001000408a, 0x4000000000000, 0x0, 0x2000000000, 0x0, 0x0, 0x0, 0x800},
	/* idx: 460 */ {0x20000001000408a, 0x8000000000000, 0x0, 0x4000000000, 0x0, 0x0, 0x0, 0x1000},
	/* idx: 461 */ {0x20000001000408a, 0x8000000000000, 0x0, 0x4000000000, 0x0, 0x0, 0x0, 0x2000},
	/* idx: 462 */ {0x20000001000408a, 0x8000000000000, 0x0, 0x8000000000, 0x0, 0x0, 0x0, 0x4000},
	/* idx: 463 */ {0x20000001000408a, 0x8000000000000, 0x0, 0x8000000000, 0x0, 0x0, 0x0, 0x8000},
	/* idx: 464 */ {0x40000002000408a, 0x10000000000000, 0x0, 0x10000000000, 0x0, 0x0, 0x0, 0x10000},
	/* idx: 465 */ {0x40000002000408a, 0x10000000000000, 0x0, 0x10000000000, 0x0, 0x0, 0x0, 0x20000},
	/* idx: 466 */ {0x40000002000408a, 0x10000000000000, 0x0, 0x20000000000, 0x0, 0x0, 0x0, 0x40000},
	/* idx: 467 */ {0x40000002000408a, 0x10000000000000, 0x0, 0x20000000000, 0x0, 0x0, 0x0, 0x80000},
	/* idx: 468 */ {0x40000002000408a, 0x20000000000000, 0x0, 0x40000000000, 0x0, 0x0, 0x0, 0x100000},
	/* idx: 469 */ {0x40000002000408a, 0x20000000000000, 0x0, 0x40000000000, 0x0, 0x0, 0x0, 0x200000},
	/* idx: 470 */ {0x40000002000408a, 0x20000000000000, 0x0, 0x80000000000, 0x0, 0x0, 0x0, 0x400000},
	/* idx: 471 */ {0x40000002000408a, 0x20000000000000, 0x0, 0x80000000000, 0x0, 0x0, 0x0, 0x800000},
	/* idx: 472 */ {0x80000002000408a, 0x40000000000000, 0x0, 0x100000000000, 0x0, 0x0, 0x0, 0x1000000},
	/* idx: 473 */ {0x80000002000408a, 0x40000000000000, 0x0, 0x100000000000, 0x0, 0x0, 0x0, 0x2000000},
	/* idx: 474 */ {0x80000002000408a, 0x40000000000000, 0x0, 0x200000000000, 0x0, 0x0, 0x0, 0x4000000},
	/* idx: 475 */ {0x80000002000408a, 0x40000000000000, 0x0, 0x200000000000, 0x0, 0x0, 0x0, 0x8000000},
	/* idx: 476 */ {0x80000002000408a, 0x80000000000000, 0x0, 0x400000000000, 0x0, 0x0, 0x0, 0x10000000},
	/* idx: 477 */ {0x80000002000408a, 0x80000000000000, 0x0, 0x400000000000, 0x0, 0x0, 0x0, 0x20000000},
	/* idx: 478 */ {0x80000002000408a, 0x80000000000000, 0x0, 0x800000000000, 0x0, 0x0, 0x0, 0x40000000},
	/* idx: 479 */ {0x80000002000408a, 0x80000000000000, 0x0, 0x800000000000, 0x0, 0x0, 0x0, 0x80000000},
	/* idx: 480 */ {0x100000004000808a, 0x100000000000000, 0x0, 0x1000000000000, 0x0, 0x0, 0x0, 0x100000000},
	/* idx: 481 */ {0x100000004000808a, 0x100000000000000, 0x0, 0x1000000000000, 0x0, 0x0, 0x0, 0x200000000},
	/* idx: 482 */ {0x100000004000808a, 0x100000000000000, 0x0, 0x2000000000000, 0x0, 0x0, 0x0, 0x400000000},
	/* idx: 483 */ {0x100000004000808a, 0x100000000000000, 0x0, 0x2000000000000, 0x0, 0x0, 0x0, 0x800000000},
	/* idx: 484 */ {0x100000004000808a, 0x200000000000000, 0x0, 0x4000000000000, 0x0, 0x0, 0x0, 0x1000000000},
	/* idx: 485 */ {0x100000004000808a, 0x200000000000000, 0x0, 0x4000000000000, 0x0, 0x0, 0x0, 0x2000000000},
	/* idx: 486 */ {0x100000004000808a, 0x200000000000000, 0x0, 0x8000000000000, 0x0, 0x0, 0x0, 0x4000000000},
	/* idx: 487 */ {0x100000004000808a, 0x200000000000000, 0x0, 0x8000000000000, 0x0, 0x0, 0x0, 0x8000000000},
	/* idx: 488 */ {0x200000004000808a, 0x400000000000000, 0x0, 0x10000000000000, 0x0, 0x0, 0x0, 0x10000000000},
	/* idx: 489 */ {0x200000004000808a, 0x400000000000000, 0x0, 0x10000000000000, 0x0, 0x0, 0x0, 0x20000000000},
	/* idx: 490 */ {0x200000004000808a, 0x400000000000000, 0x0, 0x20000000000000, 0x0, 0x0, 0x0, 0x40000000000},
	/* idx: 491 */ {0x200000004000808a, 0x400000000000000, 0x0, 0x20000000000000, 0x0, 0x0, 0x0, 0x80000000000},
	/* idx: 492 */ {0x200000004000808a, 0x800000000000000, 0x0, 0x40000000000000, 0x0, 0x0, 0x0, 0x100000000000},
	/* idx: 493 */ {0x200000004000808a, 0x800000000000000, 0x0, 0x40000000000000, 0x0, 0x0, 0x0, 0x200000000000},
	/* idx: 494 */ {0x200000004000808a, 0x800000000000000, 0x0, 0x80000000000000, 0x0, 0x0, 0x0, 0x400000000000},
	/* idx: 495 */ {0x200000004000808a, 0x800000000000000, 0x0, 0x80000000000000, 0x0, 0x0, 0x0, 0x800000000000},
	/* idx: 496 */ {0x400000008000808a, 0x1000000000000000, 0x0, 0x100000000000000, 0x0, 0x0, 0x0, 0x1000000000000},
	/* idx: 497 */ {0x400000008000808a, 0x1000000000000000, 0x0, 0x100000000000000, 0x0, 0x0, 0x0, 0x2000000000000},
	/* idx: 498 */ {0x400000008000808a, 0x1000000000000000, 0x0, 0x200000000000000, 0x0, 0x0, 0x0, 0x4000000000000},
	/* idx: 499 */ {0x400000008000808a, 0x1000000000000000, 0x0, 0x200000000000000, 0x0, 0x0, 0x0, 0x8000000000000},
	/* idx: 500 */ {0x400000008000808a, 0x2000000000000000, 0x0, 0x400000000000000, 0x0, 0x0, 0x0, 0x10000000000000},
	/* idx: 501 */ {0x400000008000808a, 0x2000000000000000, 0x0, 0x400000000000000, 0x0, 0x0, 0x0, 0x20000000000000},
	/* idx: 502 */ {0x400000008000808a, 0x2000000000000000, 0x0, 0x800000000000000, 0x0, 0x0, 0x0, 0x40000000000000},
	/* idx: 503 */ {0x400000008000808a, 0x2000000000000000, 0x0, 0x800000000000000, 0x0, 0x0, 0x0, 0x80000000000000},
	/* idx: 504 */ {0x800000008000808a, 0x4000000000000000, 0x0, 0x1000000000000000, 0x0, 0x0, 0x0, 0x100000000000000},
	/* idx: 505 */ {0x800000008000808a, 0x4000000000000000, 0x0, 0x1000000000000000, 0x0, 0x0, 0x0, 0x200000000000000},
	/* idx: 506 */ {0x800000008000808a, 0x4000000000000000, 0x0, 0x2000000000000000, 0x0, 0x0, 0x0, 0x400000000000000},
	/* idx: 507 */ {0x800000008000808a, 0x4000000000000000, 0x0, 0x2000000000000000, 0x0, 0x0, 0x0, 0x800000000000000},
	/* idx: 508 */ {0x800000008000808a, 0x8000000000000000, 0x0, 0x4000000000000000, 0x0, 0x0, 0x0, 0x1000000000000000},
	/* idx: 509 */ {0x800000008000808a, 0x8000000000000000, 0x0, 0x4000000000000000, 0x0, 0x0, 0x0, 0x2000000000000000},
	/* idx: 510 */ {0x800000008000808a, 0x8000000000000000, 0x0, 0x8000000000000000, 0x0, 0x0, 0x0, 0x4000000000000000},
	/* idx: 511 */ {0x800000008000808a, 0x8000000000000000, 0x0, 0x8000000000000000, 0x0, 0x0, 0x0, 0x8000000000000000},
}
