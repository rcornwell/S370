/*
 * Generic Card read/punch conversion tables
 *
 * Copyright (c) 2021-2024, Richard Cornwell
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 *
 */

package card

// Character conversion tables.

var asciiToHol26 = [128]uint16{
	/* Control                              */
	0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, /*0-37*/
	/*Control*/
	0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000,
	/*Control*/
	0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000,
	/*Control*/
	0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000,
	/*  sp      !      "      #      $      %      &      ' */
	/* none   Y28    78     T28    Y38    T48    XT     48  */
	0x000, 0x600, 0x006, 0x282, 0x442, 0x222, 0xA00, 0x022, /* 40 - 77 */
	/*   (      )      *      +      ,      -      .      / */
	/* T48    X48    Y48    X      T38    T      X38    T1  */
	0x222, 0x822, 0x422, 0x800, 0x242, 0x400, 0x842, 0x300,
	/*   0      1      2      3      4      5      6      7 */
	/* T      1      2      3      4      5      6      7   */
	0x200, 0x100, 0x080, 0x040, 0x020, 0x010, 0x008, 0x004,
	/*   8      9      :      ;      <      =      >      ? */
	/* 8      9      58     Y68    X68    38     68     X28 */
	0x002, 0x001, 0x012, 0x40A, 0x80A, 0x042, 0x00A, 0x882,
	/*   @      A      B      C      D      E      F      G */
	/*  82    X1     X2     X3     X4     X5     X6     X7  */
	0x022, 0x900, 0x880, 0x840, 0x820, 0x810, 0x808, 0x804, /* 100 - 137 */
	/*   H      I      J      K      L      M      N      O */
	/* X8     X9     Y1     Y2     Y3     Y4     Y5     Y6  */
	0x802, 0x801, 0x500, 0x480, 0x440, 0x420, 0x410, 0x408,
	/*   P      Q      R      S      T      U      V      W */
	/* Y7     Y8     Y9     T2     T3     T4     T5     T6  */
	0x404, 0x402, 0x401, 0x280, 0x240, 0x220, 0x210, 0x208,
	/*   X      Y      Z      [      \      ]      ^      _ */
	/* T7     T8     T9     X58    X68    T58    T78     28 */
	0x204, 0x202, 0x201, 0x812, 0x20A, 0x412, 0x406, 0x082,
	/*   `      a      b      c      d      e      f      g */
	0x212, 0xB00, 0xA80, 0xA40, 0xA20, 0xA10, 0xA08, 0xA04, /* 140 - 177 */
	/*   h      i      j      k      l      m      n      o */
	0xA02, 0xA01, 0xD00, 0xC80, 0xC40, 0xC20, 0xC10, 0xC08,
	/*   p      q      r      s      t      u      v      w */
	0xC04, 0xC02, 0xC01, 0x680, 0x640, 0x620, 0x610, 0x608,
	/*   x      y      z      {      |      }      ~    del */
	/*                     T79     X78   X79     79         */
	0x604, 0x602, 0x601, 0x406, 0x806, 0x805, 0x005, 0xf000,
}

// Set for IBM 029 codes.
var asciiToHol29 = [128]uint16{
	/* Control                              */
	0xf000, 0xf000, 0x0881, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, /*0-37*/
	/*Control*/
	0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000,
	/*Control*/
	0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000,
	/*Control*/
	0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000,
	/*  sp      !      "      #      $      %      &      ' */
	/* none   X28    78      38    Y38    T48    X      58  */
	0x000, 0x482, 0x006, 0x042, 0x442, 0x222, 0x800, 0x012, /* 40 - 77 */
	/*   (      )      *      +      ,      -      .      / */
	/* X58    Y58    Y48    X68    T38    Y      X38    T1  */
	0x812, 0x412, 0x422, 0x80A, 0x242, 0x400, 0x842, 0x300,
	/*   0      1      2      3      4      5      6      7 */
	/* T      1      2      3      4      5      6      7   */
	0x200, 0x100, 0x080, 0x040, 0x020, 0x010, 0x008, 0x004,
	/*   8      9      :      ;      <      =      >      ? */
	/* 8      9      28     Y68    X48     68    T68    T78 */
	0x002, 0x001, 0x082, 0x40A, 0x822, 0x00A, 0x20A, 0x206,
	/*   @      A      B      C      D      E      F      G */
	/*  48    X1     X2     X3     X4     X5     X6     X7  */
	0x022, 0x900, 0x880, 0x840, 0x820, 0x810, 0x808, 0x804, /* 100 - 137 */
	/*   H      I      J      K      L      M      N      O */
	/* X8     X9     Y1     Y2     Y3     Y4     Y5     Y6  */
	0x802, 0x801, 0x500, 0x480, 0x440, 0x420, 0x410, 0x408,
	/*   P      Q      R      S      T      U      V      W */
	/* Y7     Y8     Y9     T2     T3     T4     T5     T6  */
	0x404, 0x402, 0x401, 0x280, 0x240, 0x220, 0x210, 0x208,
	/*   X      Y      Z      [      \      ]      ^      _ */
	/* T7     T8     T9   TY028    T28  TY038    Y78    T58 */
	0x204, 0x202, 0x201, 0xE82, 0x282, 0xE42, 0x406, 0x212,
	/*   `      a      b      c      d      e      f      g */
	0x102, 0xB00, 0xA80, 0xA40, 0xA20, 0xA10, 0xA08, 0xA04, /* 140 - 177 */
	/*   h      i      j      k      l      m      n      o */
	0xA02, 0xA01, 0xD00, 0xC80, 0xC40, 0xC20, 0xC10, 0xC08,
	/*   p      q      r      s      t      u      v      w */
	0xC04, 0xC02, 0xC01, 0x680, 0x640, 0x620, 0x610, 0x608,
	/*   x      y      z      {      |      }      ~    del */
	/*                      Y78    X78    X79  XTY18        */
	0x604, 0x602, 0x601, 0x406, 0x806, 0x805, 0xF02, 0xf000,
}

// Set for IBM DEC 029 codes.
var asciiToDecHol29 = [128]uint16{
	/* Control                              */
	0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, /*0-37*/
	/*Control*/
	0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000,
	/*Control*/
	0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000,
	/*Control*/
	0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000,
	/*  sp      !      "      #      $      %      &      ' */
	/* none   Y28    78      38    Y38    T48    X      58  */
	0x000, 0x482, 0x006, 0x042, 0x442, 0x222, 0x800, 0x012, /* 40 - 77 */
	/*   (      )      *      +      ,      -      .      / */
	/* X58    Y58    Y48    X68    T38    Y      X38    T1  */
	0x812, 0x412, 0x422, 0x80A, 0x242, 0x400, 0x842, 0x300,
	/*   0      1      2      3      4      5      6      7 */
	/* T      1      2      3      4      5      6      7   */
	0x200, 0x100, 0x080, 0x040, 0x020, 0x010, 0x008, 0x004,
	/*   8      9      :      ;      <      =      >      ? */
	/* 8      9      28     Y68    X48     68    T68    T78 */
	0x002, 0x001, 0x082, 0x40A, 0x822, 0x00A, 0x20A, 0x206,
	/*   @      A      B      C      D      E      F      G */
	/*  48    X1     X2     X3     X4     X5     X6     X7  */
	0x022, 0x900, 0x880, 0x840, 0x820, 0x810, 0x808, 0x804, /* 100 - 137 */
	/*   H      I      J      K      L      M      N      O */
	/* X8     X9     Y1     Y2     Y3     Y4     Y5     Y6  */
	0x802, 0x801, 0x500, 0x480, 0x440, 0x420, 0x410, 0x408,
	/*   P      Q      R      S      T      U      V      W */
	/* Y7     Y8     Y9     T2     T3     T4     T5     T6  */
	0x404, 0x402, 0x401, 0x280, 0x240, 0x220, 0x210, 0x208,
	/*   X      Y      Z      [      \      ]      ^      _ */
	/* T7     T8     T9     X28    Y78    T28    X78    T58 */
	0x204, 0x202, 0x201, 0x882, 0x406, 0x282, 0x806, 0x212,
	/*   `      a      b      c      d      e      f      g */
	0x102, 0xB00, 0xA80, 0xA40, 0xA20, 0xA10, 0xA08, 0xA04, /* 140 - 177 */
	/*   h      i      j      k      l      m      n      o */
	0xA02, 0xA01, 0xD00, 0xC80, 0xC40, 0xC20, 0xC10, 0xC08,
	/*   p      q      r      s      t      u      v      w */
	0xC04, 0xC02, 0xC01, 0x680, 0x640, 0x620, 0x610, 0x608,
	/*   x      y      z      {      |      }      ~    del */
	/*                       XT     XY     YT    YT1        */
	0x604, 0x602, 0x601, 0xA00, 0xC00, 0x600, 0x700, 0xf000,
}

// Ascii codes to IBM EBCDIC punch codes.
var asciiToHolEbcdic = [128]uint16{
	/* Control                              */
	0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, /*0-37*/
	/*Control*/
	0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000,
	/*Control*/
	0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000,
	/*Control*/
	0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000, 0xf000,
	/*  sp      !      "      #      $      %      &      ' */
	/* none   Y28    78      38    Y38    T48    X      58  */
	0x000, 0x482, 0x006, 0x042, 0x442, 0x222, 0x800, 0x012, /* 40 - 77 */
	/*   (      )      *      +      ,      -      .      / */
	/* X58    Y58    Y48    X      T38    Y      X38    T1  */
	0x812, 0x412, 0x422, 0x800, 0x242, 0x400, 0x842, 0x300,
	/*   0      1      2      3      4      5      6      7 */
	/* T      1      2      3      4      5      6      7   */
	0x200, 0x100, 0x080, 0x040, 0x020, 0x010, 0x008, 0x004,
	/*   8      9      :      ;      <      =      >      ? */
	/* 8      9      28     Y68    X48    68     T68    T78 */
	0x002, 0x001, 0x082, 0x40A, 0x822, 0x00A, 0x20A, 0x206,
	/*   @      A      B      C      D      E      F      G */
	/*  48    X1     X2     X3     X4     X5     X6     X7  */
	0x022, 0x900, 0x880, 0x840, 0x820, 0x810, 0x808, 0x804, /* 100 - 137 */
	/*   H      I      J      K      L      M      N      O */
	/* X8     X9     Y1     Y2     Y3     Y4     Y5     Y6  */
	0x802, 0x801, 0x500, 0x480, 0x440, 0x420, 0x410, 0x408,
	/*   P      Q      R      S      T      U      V      W */
	/* Y7     Y8     Y9     T2     T3     T4     T5     T6  */
	0x404, 0x402, 0x401, 0x280, 0x240, 0x220, 0x210, 0x208,
	/*   X      Y      Z      [      \      ]      ^      _ */
	/* T7     T8     T9     X28    X68    Y28    Y78    X58 */
	0x204, 0x202, 0x201, 0x882, 0x20A, 0x482, 0x406, 0x212,
	/*   `      a      b      c      d      e      f      g */
	0x102, 0xB00, 0xA80, 0xA40, 0xA20, 0xA10, 0xA08, 0xA04, /* 140 - 177 */
	/*   h      i      j      k      l      m      n      o */
	0xA02, 0xA01, 0xD00, 0xC80, 0xC40, 0xC20, 0xC10, 0xC08,
	/*   p      q      r      s      t      u      v      w */
	0xC04, 0xC02, 0xC01, 0x680, 0x640, 0x620, 0x610, 0x608,
	/*   x      y      z      {      |      }      ~    del */
	/*                     X18     X78    Y18  XYT18        */
	0x604, 0x602, 0x601, 0x902, 0x806, 0x502, 0xF02, 0xf000,
}

// IBM EBCDIC codes to IBM punch codes.
var ebcdicToHolTable = [256]uint16{
	/*  T918    T91    T92    T93    T94    T95    T96   T97   0x0x */
	0xB03, 0x901, 0x881, 0x841, 0x821, 0x811, 0x809, 0x805,
	/*  T98,   T189 , T289,  T389,  T489,  T589,  T689, T789   */
	0x803, 0x903, 0x883, 0x843, 0x823, 0x813, 0x80B, 0x807,
	/* TE189    E91    E92    E93    E94    E95    E96   E97   0x1x */
	0xD03, 0x501, 0x481, 0x441, 0x421, 0x411, 0x409, 0x405,
	/*  E98     E918   E928   E938   E948   E958   E968  E978   */
	0x403, 0x503, 0x483, 0x443, 0x423, 0x413, 0x40B, 0x407,
	/*  E0918   091    092    093    094    095    096   097   0x2x */
	0x703, 0x301, 0x281, 0x241, 0x221, 0x211, 0x209, 0x205,
	/*  098     0918  0928   0938    0948   0958   0968  0978   */
	0x203, 0x303, 0x283, 0x243, 0x223, 0x213, 0x20B, 0x207,
	/* TE0918   91    92     93      94     95     96     97   0x3x */
	0xF03, 0x101, 0x081, 0x041, 0x021, 0x011, 0x009, 0x005,
	/*  98      189    289    389    489    589    689    789   */
	0x003, 0x103, 0x083, 0x043, 0x023, 0x013, 0x00B, 0x007,
	/*          T091  T092   T093   T094   T095   T096    T097  0x4x */
	0x000, 0xB01, 0xA81, 0xA41, 0xA21, 0xA11, 0xA09, 0xA05,
	/* T098     T18    T28    T38    T48    T58    T68    T78    */
	0xA03, 0x902, 0x882, 0x842, 0x822, 0x812, 0x80A, 0x806,
	/* T        TE91  TE92   TE93   TE94   TE95   TE96    TE97  0x5x */
	0x800, 0xD01, 0xC81, 0xC41, 0xC21, 0xC11, 0xC09, 0xC05,
	/* TE98     E18    E28    E38    E48    E58    E68    E78   */
	0xC03, 0x502, 0x482, 0x442, 0x422, 0x412, 0x40A, 0x406,
	/* E        01    E092   E093   E094   E095   E096    E097  0x6x */
	0x400, 0x300, 0x681, 0x641, 0x621, 0x611, 0x609, 0x605,
	/* E098     018   TE     038    048     68    068     078    */
	0x603, 0x302, 0xC00, 0x242, 0x222, 0x212, 0x20A, 0x206,
	/* TE0    TE091  TE092  TE093  TE094  TE095  TE096  TE097   0x7x */
	0xE00, 0xF01, 0xE81, 0xE41, 0xE21, 0xE11, 0xE09, 0xE05,
	/* TE098    18     28     38    48      58      68     78    */
	0xE03, 0x102, 0x082, 0x042, 0x022, 0x012, 0x00A, 0x006,
	/* T018     T01    T02    T03    T04    T05    T06    T07   0x8x */
	0xB02, 0xB00, 0xA80, 0xA40, 0xA20, 0xA10, 0xA08, 0xA04,
	/* T08      T09   T028   T038    T048   T058   T068   T078   */
	0xA02, 0xA01, 0xA82, 0xA42, 0xA22, 0xA12, 0xA0A, 0xA06,
	/* TE18     TE1    TE2    TE3    TE4    TE5    TE6    TE7   0x9x */
	0xD02, 0xD00, 0xC80, 0xC40, 0xC20, 0xC10, 0xC08, 0xC04,
	/* TE8      TE9   TE28   TE38    TE48   TE58   TE68   TE78   */
	0xC02, 0xC01, 0xC82, 0xC42, 0xC22, 0xC12, 0xC0A, 0xC06,
	/* E018     E01    E02    E03    E04    E05    E06    E07   0xax */
	0x702, 0x700, 0x680, 0x640, 0x620, 0x610, 0x608, 0x604,
	/* E08      E09   E028   E038    E048   E058   E068   E078   */
	0x602, 0x601, 0x682, 0x642, 0x622, 0x612, 0x60A, 0x606,
	/* TE018    TE01   TE02   TE03   TE04   TE05   TE06   TE07  0xbx */
	0xF02, 0xF00, 0xE80, 0xE40, 0xE20, 0xE10, 0xE08, 0xE04,
	/* TE08     TE09   TE028  TE038  TE048  TE058  TE068  TE078  */
	0xE02, 0xE01, 0xE82, 0xE42, 0xE22, 0xE12, 0xE0A, 0xE06,
	/*  T0      T1     T2     T3     T4     T5     T6     T7    0xcx */
	0xA00, 0x900, 0x880, 0x840, 0x820, 0x810, 0x808, 0x804,
	/* T8       T9     T0928  T0938  T0948  T0958  T0968  T0978  */
	0x802, 0x801, 0xA83, 0xA43, 0xA23, 0xA13, 0xA0B, 0xA07,
	/* E0       E1     E2     E3     E4     E5     E6     E7    0xdx */
	0x600, 0x500, 0x480, 0x440, 0x420, 0x410, 0x408, 0x404,
	/* E8       E9     TE928  TE938  TE948  TE958  TE968  TE978  */
	0x402, 0x401, 0xC83, 0xC43, 0xC23, 0xC13, 0xC0B, 0xC07,
	/* 028      E091   02     03     04     05     06     07    0xex  */
	0x282, 0x701, 0x280, 0x240, 0x220, 0x210, 0x208, 0x204,
	/* 08       09     E0928  E0938  E0948  E0958  E0968  E0978  */
	0x202, 0x201, 0x683, 0x643, 0x623, 0x613, 0x60B, 0x607,
	/* 0        1      2      3      4      5      6      7     0xfx */
	0x200, 0x100, 0x080, 0x040, 0x020, 0x010, 0x008, 0x004,
	/* 8        9     TE0928 TE0938 TE0948 TE0958 TE0968 TE0978  */
	0x002, 0x001, 0xE83, 0xE43, 0xE23, 0xE13, 0xE0B, 0xE07,
}

var asciiToSix = [128]int8{
	/* Control                              */
	-1, -1, -1, -1, -1, -1, -1, -1, /* 0 - 37 */
	/* Control                              */
	-1, -1, -1, -1, -1, -1, -1, -1,
	/* Control                              */
	-1, -1, -1, -1, -1, -1, -1, -1,
	/* Control                              */
	-1, -1, -1, -1, -1, -1, -1, -1,
	/*sp    !    "    #    $    %    &    ' */
	0o00, 0o52, -1, 0o32, 0o53, 0o17, 0o60, 0o14, /* 40 - 77 */
	/* (    )    *    +    ,    -    .    / */
	0o34, 0o74, 0o54, 0o60, 0o33, 0o40, 0o73, 0o21,
	/* 0    1    2    3    4    5    6    7 */
	0o12, 0o01, 0o02, 0o03, 0o04, 0o05, 0o06, 0o07,
	/* 8    9    :    ;    <    =    >    ? */
	0o10, 0o11, 0o15, 0o56, 0o76, 0o13, 0o16, 0o72,
	/* @    A    B    C    D    E    F    G */
	0o14, 0o61, 0o62, 0o63, 0o64, 0o65, 0o66, 0o67, /* 100 - 137 */
	/* H    I    J    K    L    M    N    O */
	0o70, 0o71, 0o41, 0o42, 0o43, 0o44, 0o45, 0o46,
	/* P    Q    R    S    T    U    V    W */
	0o47, 0o50, 0o51, 0o22, 0o23, 0o24, 0o25, 0o26,
	/* X    Y    Z    [    \    ]    ^    _ */
	0o27, 0o30, 0o31, 0o75, 0o36, 0o55, 0o57, 0o20,
	/* `    a    b    c    d    e    f    g */
	0o35, 0o61, 0o62, 0o63, 0o64, 0o65, 0o66, 0o67, /* 140 - 177 */
	/* h    i    j    k    l    m    n    o */
	0o70, 0o71, 0o41, 0o42, 0o43, 0o44, 0o45, 0o46,
	/* p    q    r    s    t    u    v    w */
	0o47, 0o50, 0o51, 0o22, 0o23, 0o24, 0o25, 0o26,
	/* x    y    z    {    |    }    ~   del*/
	0o27, 0o30, 0o31, 0o57, 0o77, 0o17, -1, -1,
}

// Back tables which are automatically generated.
var (
	holToASCIITable26 [4096]uint8
	holToASCIITable29 [4096]uint8
	holToEBCDICTable  [4096]uint16
)
