github.com/scttfrdmn/keel/internal/l1.avx512Asum STEXT nosplit size=399 align=0x0 args=0x18 locals=0x18 funcid=0x0
	0x0000 00000 (internal/l1.avx512Asum(SB), NOSPLIT|ABIInternal, $24-24
	0x0000 00000 (internal/l1/l1_amd64.go:227)	PUSHQ	BP
	0x0001 00001 (internal/l1/l1_amd64.go:227)	MOVQ	SP, BP
	0x0004 00004 (internal/l1/l1_amd64.go:227)	SUBQ	$16, SP
	0x0008 00008 (internal/l1/l1_amd64.go:227)	MOVQ	AX, github.com/scttfrdmn/keel/internal/l1.x+32(FP)
	0x000d 00013 (internal/l1/l1_amd64.go:227)	FUNCDATA	$0, gclocals·wvjpxkknJ4nY1JtrArJJaw==(SB)
	0x000d 00013 (internal/l1/l1_amd64.go:227)	FUNCDATA	$1, gclocals·J26BEvPExEQhJvjp9E8Whg==(SB)
	0x000d 00013 (internal/l1/l1_amd64.go:227)	FUNCDATA	$5, github.com/scttfrdmn/keel/internal/l1.avx512Asum.arginfo1(SB)
	0x000d 00013 (internal/l1/l1_amd64.go:227)	FUNCDATA	$6, github.com/scttfrdmn/keel/internal/l1.avx512Asum.argliveinfo(SB)
	0x000d 00013 (internal/l1/l1_amd64.go:227)	PCDATA	$3, $1
	0x000d 00013 (/Volumes/External HDSIMD/archsimd/other_gen_amd64.go:211)	MOVL	$2147483647, DX
	0x0012 00018 (/Volumes/External HDSIMD/archsimd/other_gen_amd64.go:211)	VMOVD	DX, X1
	0x0016 00022 (internal/l1/l1_amd64.go:228)	XCHGL	AX, AX
	0x0017 00023 (internal/l1/l1_amd64.go:232)	XCHGL	AX, AX
	0x0018 00024 (internal/vec/vec_avx512.go:56)	VPXORQ	Z2, Z2, Z2
	0x001e 00030 (internal/vec/vec_avx512.go:128)	XCHGL	AX, AX
	0x001f 00031 (/Volumes/External HDSIMD/archsimd/other_gen_amd64.go:211)	VPBROADCASTD	X1, Z1
	0x0025 00037 (internal/l1/l1_amd64.go:233)	VMOVDQU64	Z2, Z0
	0x002b 00043 (internal/l1/l1_amd64.go:233)	VMOVDQU64	Z0, Z3
	0x0031 00049 (internal/l1/l1_amd64.go:233)	VMOVDQU64	Z3, Z4
	0x0037 00055 (internal/l1/l1_amd64.go:233)	JMP	130
	0x0039 00057 (internal/vec/vec_avx512.go:154)	VPANDD	(AX), Z1, Z5
	0x003f 00063 (internal/vec/vec_avx512.go:154)	VPANDD	64(AX), Z1, Z6
	0x0046 00070 (internal/vec/vec_avx512.go:154)	VPANDD	128(AX), Z1, Z7
	0x004d 00077 (internal/vec/vec_avx512.go:154)	VPANDD	192(AX), Z1, Z8
	0x0054 00084 (internal/l1/l1_amd64.go:234)	XCHGL	AX, AX
	0x0055 00085 (internal/l1/l1_amd64.go:234)	XCHGL	AX, AX
	0x0056 00086 (internal/l1/l1_amd64.go:235)	XCHGL	AX, AX
	0x0057 00087 (internal/l1/l1_amd64.go:235)	XCHGL	AX, AX
	0x0058 00088 (internal/l1/l1_amd64.go:236)	XCHGL	AX, AX
	0x0059 00089 (internal/l1/l1_amd64.go:236)	XCHGL	AX, AX
	0x005a 00090 (internal/l1/l1_amd64.go:237)	XCHGL	AX, AX
	0x005b 00091 (internal/l1/l1_amd64.go:237)	XCHGL	AX, AX
	0x005c 00092 (internal/vec/vec_avx512.go:59)	VADDPS	Z5, Z3, Z3
	0x0062 00098 (internal/vec/vec_avx512.go:59)	VADDPS	Z6, Z4, Z4
	0x0068 00104 (internal/vec/vec_avx512.go:59)	VADDPS	Z7, Z0, Z0
	0x006e 00110 (internal/vec/vec_avx512.go:59)	VADDPS	Z8, Z2, Z2
	0x0074 00116 (internal/l1/l1_amd64.go:238)	ADDQ	$-64, CX
	0x0078 00120 (internal/l1/l1_amd64.go:238)	ADDQ	$256, AX
	0x007e 00126 (internal/l1/l1_amd64.go:238)	ADDQ	$-64, BX
	0x0082 00130 (internal/l1/l1_amd64.go:233)	CMPQ	BX, $64
	0x0086 00134 (internal/l1/l1_amd64.go:233)	JGT	57
	0x0088 00136 (internal/l1/l1_amd64.go:240)	JNE	239
	0x008a 00138 (internal/vec/vec_avx512.go:154)	VPANDD	(AX), Z1, Z5
	0x0090 00144 (internal/vec/vec_avx512.go:154)	VPANDD	64(AX), Z1, Z6
	0x0097 00151 (internal/vec/vec_avx512.go:154)	VPANDD	128(AX), Z1, Z7
	0x009e 00158 (internal/vec/vec_avx512.go:154)	VPANDD	192(AX), Z1, Z8
	0x00a5 00165 (internal/l1/l1_amd64.go:241)	XCHGL	AX, AX
	0x00a6 00166 (internal/l1/l1_amd64.go:241)	XCHGL	AX, AX
	0x00a7 00167 (internal/l1/l1_amd64.go:242)	XCHGL	AX, AX
	0x00a8 00168 (internal/l1/l1_amd64.go:242)	XCHGL	AX, AX
	0x00a9 00169 (internal/l1/l1_amd64.go:243)	XCHGL	AX, AX
	0x00aa 00170 (internal/l1/l1_amd64.go:243)	XCHGL	AX, AX
	0x00ab 00171 (internal/l1/l1_amd64.go:244)	XCHGL	AX, AX
	0x00ac 00172 (internal/l1/l1_amd64.go:244)	XCHGL	AX, AX
	0x00ad 00173 (internal/vec/vec_avx512.go:59)	VADDPS	Z5, Z3, Z3
	0x00b3 00179 (internal/vec/vec_avx512.go:59)	VADDPS	Z6, Z4, Z4
	0x00b9 00185 (internal/vec/vec_avx512.go:59)	VADDPS	Z7, Z0, Z0
	0x00bf 00191 (internal/vec/vec_avx512.go:59)	VADDPS	Z8, Z2, Z2
	0x00c5 00197 (internal/vec/vec_avx512.go:59)	XORL	BX, BX
	0x00c7 00199 (internal/l1/l1_amd64.go:245)	JMP	239
	0x00c9 00201 (internal/l1/l1_amd64.go:249)	ADDQ	$-16, CX
	0x00cd 00205 (internal/l1/l1_amd64.go:249)	MOVQ	CX, DX
	0x00d0 00208 (internal/l1/l1_amd64.go:249)	NEGQ	DX
	0x00d3 00211 (internal/l1/l1_amd64.go:249)	SARQ	$63, DX
	0x00d7 00215 (internal/l1/l1_amd64.go:249)	ANDL	$64, DX
	0x00da 00218 (internal/vec/vec_avx512.go:154)	VPANDD	(AX), Z1, Z5
	0x00e0 00224 (internal/l1/l1_amd64.go:248)	XCHGL	AX, AX
	0x00e1 00225 (internal/l1/l1_amd64.go:248)	XCHGL	AX, AX
	0x00e2 00226 (internal/l1/l1_amd64.go:249)	ADDQ	DX, AX
	0x00e5 00229 (internal/vec/vec_avx512.go:59)	VADDPS	Z5, Z3, Z3
	0x00eb 00235 (internal/l1/l1_amd64.go:249)	ADDQ	$-16, BX
	0x00ef 00239 (internal/l1/l1_amd64.go:247)	CMPQ	BX, $16
	0x00f3 00243 (internal/l1/l1_amd64.go:247)	JGE	201
	0x00f5 00245 (internal/l1/l1_amd64.go:251)	TESTQ	BX, BX
	0x00f8 00248 (internal/l1/l1_amd64.go:251)	JEQ	298
	0x00fa 00250 (/Volumes/External HDSIMD/archsimd/slice_gen_amd64.go:654)	LEAQ	-16(BX), CX
	0x00fe 00254 (/Volumes/External HDSIMD/archsimd/slice_gen_amd64.go:654)	NEGQ	CX
	0x0101 00257 (/Volumes/External HDSIMD/archsimd/slice_gen_amd64.go:654)	MOVL	$-1, DX
	0x0106 00262 (/Volumes/External HDSIMD/archsimd/slice_gen_amd64.go:654)	SHRW	CX, DX
	0x0109 00265 (/Volumes/External HDSIMD/archsimd/slice_gen_amd64.go:654)	KMOVW	DX, K1
	0x010d 00269 (/Volumes/External HDSIMD/archsimd/slice_gen_amd64.go:654)	VPMOVM2D	K1, Z5
	0x0113 00275 (/Volumes/External HDSIMD/archsimd/slice_gen_amd64.go:655)	VMOVDQU64	(AX), Z6
	0x0119 00281 (internal/vec/vec_avx512.go:154)	VPTERNLOGD	$128, Z1, Z6, Z5
	0x0120 00288 (internal/l1/l1_amd64.go:252)	XCHGL	AX, AX
	0x0121 00289 (internal/l1/l1_amd64.go:252)	XCHGL	AX, AX
	0x0122 00290 (internal/l1/l1_amd64.go:252)	XCHGL	AX, AX
	0x0123 00291 (internal/vec/vec_avx512.go:44)	XCHGL	AX, AX
	0x0124 00292 (internal/vec/vec_avx512.go:59)	VADDPS	Z5, Z3, Z3
	0x012a 00298 (internal/vec/vec_avx512.go:59)	VADDPS	Z4, Z3, Z1
	0x0130 00304 (internal/vec/vec_avx512.go:59)	VADDPS	Z2, Z0, Z2
	0x0136 00310 (internal/vec/vec_avx512.go:59)	VADDPS	Z2, Z1, Z1
	0x013c 00316 (internal/vec/vec_avx512.go:166)	VEXTRACTF64X4	$0, Z1, Y2
	0x0143 00323 (internal/vec/vec_avx512.go:166)	VEXTRACTF64X4	$1, Z1, Y1
	0x014a 00330 (internal/vec/vec_avx512.go:166)	VADDPS	Y1, Y2, Y1
	0x014e 00334 (internal/vec/vec_avx512.go:167)	VEXTRACTF128	$0, Y1, X2
	0x0154 00340 (internal/vec/vec_avx512.go:167)	VEXTRACTF128	$1, Y1, X1
	0x015a 00346 (internal/vec/vec_avx512.go:167)	VADDPS	X1, X2, X1
	0x015e 00350 (<unknown line number>)	NOP
	0x015e 00350 (<unknown line number>)	NOP
	0x015e 00350 (<unknown line number>)	NOP
	0x015e 00350 (<unknown line number>)	NOP
	0x015e 00350 (internal/vec/vec_avx512.go:169)	VMOVDQU	X1, github.com/scttfrdmn/keel/internal/vec.a(SP)
	0x0163 00355 (internal/vec.a+8(SP), X0
	0x0169 00361 (internal/vec.a(SP), X0
	0x016e 00366 (internal/vec/vec_avx512.go:170)	MOVSS	X0, github.com/scttfrdmn/keel/internal/vec.a(SP)
	0x0173 00371 (internal/vec.a+12(SP), X1
	0x0179 00377 (internal/vec.a+4(SP), X1
	0x017f 00383 (internal/vec/vec_avx512.go:171)	MOVSS	X1, github.com/scttfrdmn/keel/internal/vec.a+4(SP)
	0x0185 00389 (internal/vec/vec_avx512.go:172)	ADDSS	X1, X0
	0x0189 00393 (internal/l1/l1_amd64.go:254)	ADDQ	$16, SP
	0x018d 00397 (internal/l1/l1_amd64.go:254)	POPQ	BP
	0x018e 00398 (internal/l1/l1_amd64.go:254)	RET
	0x0000 55 48 89 e5 48 83 ec 10 48 89 44 24 20 ba ff ff  UH..H...H.D$ ...
	0x0010 ff 7f c5 f9 6e ca 90 90 62 f1 ed 48 ef d2 90 62  ....n...b..H...b
	0x0020 f2 7d 48 58 c9 62 f1 fe 48 7f d0 62 f1 fe 48 7f  .}HX.b..H..b..H.
	0x0030 c3 62 f1 fe 48 7f dc eb 49 62 f1 75 48 db 28 62  .b..H...Ib.uH.(b
	0x0040 f1 75 48 db 70 01 62 f1 75 48 db 78 02 62 71 75  .uH.p.b.uH.x.bqu
	0x0050 48 db 40 03 90 90 90 90 90 90 90 90 62 f1 64 48  H.@.........b.dH
	0x0060 58 dd 62 f1 5c 48 58 e6 62 f1 7c 48 58 c7 62 d1  X.b.\HX.b.|HX.b.
	0x0070 6c 48 58 d0 48 83 c1 c0 48 05 00 01 00 00 48 83  lHX.H...H.....H.
	0x0080 c3 c0 48 83 fb 40 7f b1 75 65 62 f1 75 48 db 28  ..H..@..ueb.uH.(
	0x0090 62 f1 75 48 db 70 01 62 f1 75 48 db 78 02 62 71  b.uH.p.b.uH.x.bq
	0x00a0 75 48 db 40 03 90 90 90 90 90 90 90 90 62 f1 64  uH.@.........b.d
	0x00b0 48 58 dd 62 f1 5c 48 58 e6 62 f1 7c 48 58 c7 62  HX.b.\HX.b.|HX.b
	0x00c0 d1 6c 48 58 d0 31 db eb 26 48 83 c1 f0 48 89 ca  .lHX.1..&H...H..
	0x00d0 48 f7 da 48 c1 fa 3f 83 e2 40 62 f1 75 48 db 28  H..H..?..@b.uH.(
	0x00e0 90 90 48 01 d0 62 f1 64 48 58 dd 48 83 c3 f0 48  ..H..b.dHX.H...H
	0x00f0 83 fb 10 7d d4 48 85 db 74 30 48 8d 4b f0 48 f7  ...}.H..t0H.K.H.
	0x0100 d9 ba ff ff ff ff 66 d3 ea c5 f8 92 ca 62 f2 7e  ......f......b.~
	0x0110 48 38 e9 62 f1 fe 48 6f 30 62 f3 4d 48 25 e9 80  H8.b..Ho0b.MH%..
	0x0120 90 90 90 90 62 f1 64 48 58 dd 62 f1 64 48 58 cc  ....b.dHX.b.dHX.
	0x0130 62 f1 7c 48 58 d2 62 f1 74 48 58 ca 62 f3 fd 48  b.|HX.b.tHX.b..H
	0x0140 1b ca 00 62 f3 fd 48 1b c9 01 c5 ec 58 c9 c4 e3  ...b..H.....X...
	0x0150 7d 19 ca 00 c4 e3 7d 19 c9 01 c5 e8 58 c9 c5 fa  }.....}.....X...
	0x0160 7f 0c 24 f3 0f 10 44 24 08 f3 0f 58 04 24 f3 0f  ..$...D$...X.$..
	0x0170 11 04 24 f3 0f 10 4c 24 0c f3 0f 58 4c 24 04 f3  ..$...L$...XL$..
	0x0180 0f 11 4c 24 04 f3 0f 58 c1 48 83 c4 10 5d c3     ..L$...X.H...].
