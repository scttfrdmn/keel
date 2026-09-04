github.com/scttfrdmn/keel/internal/l1.avx512Asum STEXT nosplit size=461 align=0x0 args=0x18 locals=0x18 funcid=0x0
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
	0x000d 00013 (internal/l1/l1_amd64.go:228)	XCHGL	AX, AX
	0x000e 00014 (internal/vec/vec_avx512.go:56)	VPXORQ	Z1, Z1, Z1
	0x0014 00020 (internal/l1/l1_amd64.go:229)	VMOVDQU64	Z1, Z0
	0x001a 00026 (internal/l1/l1_amd64.go:229)	VMOVDQU64	Z0, Z2
	0x0020 00032 (internal/l1/l1_amd64.go:229)	VMOVDQU64	Z2, Z3
	0x0026 00038 (internal/l1/l1_amd64.go:229)	JMP	134
	0x0028 00040 (/Volumes/External HDSIMD/archsimd/other_gen_amd64.go:211)	MOVL	$2147483647, DX
	0x002d 00045 (/Volumes/External HDSIMD/archsimd/other_gen_amd64.go:211)	VMOVD	DX, X4
	0x0031 00049 (/Volumes/External HDSIMD/archsimd/other_gen_amd64.go:211)	VPBROADCASTD	X4, Z4
	0x0037 00055 (internal/vec/vec_avx512.go:154)	VPANDD	(AX), Z4, Z5
	0x003d 00061 (internal/vec/vec_avx512.go:154)	VPANDD	64(AX), Z4, Z6
	0x0044 00068 (internal/vec/vec_avx512.go:154)	VPANDD	128(AX), Z4, Z7
	0x004b 00075 (internal/vec/vec_avx512.go:154)	VPANDD	192(AX), Z4, Z4
	0x0052 00082 (internal/l1/l1_amd64.go:230)	XCHGL	AX, AX
	0x0053 00083 (internal/l1/l1_amd64.go:230)	XCHGL	AX, AX
	0x0054 00084 (internal/l1/l1_amd64.go:231)	XCHGL	AX, AX
	0x0055 00085 (internal/l1/l1_amd64.go:231)	XCHGL	AX, AX
	0x0056 00086 (internal/l1/l1_amd64.go:232)	XCHGL	AX, AX
	0x0057 00087 (internal/l1/l1_amd64.go:232)	XCHGL	AX, AX
	0x0058 00088 (internal/l1/l1_amd64.go:233)	XCHGL	AX, AX
	0x0059 00089 (internal/l1/l1_amd64.go:233)	XCHGL	AX, AX
	0x005a 00090 (internal/vec/vec_avx512.go:112)	XCHGL	AX, AX
	0x005b 00091 (internal/vec/vec_avx512.go:112)	XCHGL	AX, AX
	0x005c 00092 (internal/vec/vec_avx512.go:59)	VADDPS	Z5, Z2, Z2
	0x0062 00098 (internal/vec/vec_avx512.go:112)	XCHGL	AX, AX
	0x0063 00099 (internal/vec/vec_avx512.go:59)	VADDPS	Z6, Z3, Z3
	0x0069 00105 (internal/vec/vec_avx512.go:112)	XCHGL	AX, AX
	0x006a 00106 (internal/vec/vec_avx512.go:59)	VADDPS	Z7, Z0, Z0
	0x0070 00112 (internal/vec/vec_avx512.go:112)	XCHGL	AX, AX
	0x0071 00113 (internal/vec/vec_avx512.go:59)	VADDPS	Z4, Z1, Z1
	0x0077 00119 (internal/vec/vec_avx512.go:128)	XCHGL	AX, AX
	0x0078 00120 (internal/l1/l1_amd64.go:234)	ADDQ	$-64, CX
	0x007c 00124 (internal/l1/l1_amd64.go:234)	ADDQ	$256, AX
	0x0082 00130 (internal/l1/l1_amd64.go:234)	ADDQ	$-64, BX
	0x0086 00134 (internal/l1/l1_amd64.go:229)	CMPQ	BX, $64
	0x008a 00138 (internal/l1/l1_amd64.go:229)	JGT	40
	0x008c 00140 (internal/l1/l1_amd64.go:236)	JNE	226
	0x008e 00142 (/Volumes/External HDSIMD/archsimd/other_gen_amd64.go:211)	MOVL	$2147483647, DX
	0x0093 00147 (/Volumes/External HDSIMD/archsimd/other_gen_amd64.go:211)	VMOVD	DX, X4
	0x0097 00151 (/Volumes/External HDSIMD/archsimd/other_gen_amd64.go:211)	VPBROADCASTD	X4, Z4
	0x009d 00157 (internal/vec/vec_avx512.go:154)	VPANDD	(AX), Z4, Z5
	0x00a3 00163 (internal/vec/vec_avx512.go:154)	VPANDD	64(AX), Z4, Z6
	0x00aa 00170 (internal/vec/vec_avx512.go:154)	VPANDD	128(AX), Z4, Z7
	0x00b1 00177 (internal/vec/vec_avx512.go:154)	VPANDD	192(AX), Z4, Z4
	0x00b8 00184 (internal/l1/l1_amd64.go:237)	XCHGL	AX, AX
	0x00b9 00185 (internal/l1/l1_amd64.go:237)	XCHGL	AX, AX
	0x00ba 00186 (internal/l1/l1_amd64.go:238)	XCHGL	AX, AX
	0x00bb 00187 (internal/l1/l1_amd64.go:238)	XCHGL	AX, AX
	0x00bc 00188 (internal/l1/l1_amd64.go:239)	XCHGL	AX, AX
	0x00bd 00189 (internal/l1/l1_amd64.go:239)	XCHGL	AX, AX
	0x00be 00190 (internal/l1/l1_amd64.go:240)	XCHGL	AX, AX
	0x00bf 00191 (internal/l1/l1_amd64.go:240)	XCHGL	AX, AX
	0x00c0 00192 (internal/vec/vec_avx512.go:112)	XCHGL	AX, AX
	0x00c1 00193 (internal/vec/vec_avx512.go:112)	XCHGL	AX, AX
	0x00c2 00194 (internal/vec/vec_avx512.go:59)	VADDPS	Z5, Z2, Z2
	0x00c8 00200 (internal/vec/vec_avx512.go:112)	XCHGL	AX, AX
	0x00c9 00201 (internal/vec/vec_avx512.go:59)	VADDPS	Z6, Z3, Z3
	0x00cf 00207 (internal/vec/vec_avx512.go:112)	XCHGL	AX, AX
	0x00d0 00208 (internal/vec/vec_avx512.go:59)	VADDPS	Z7, Z0, Z0
	0x00d6 00214 (internal/vec/vec_avx512.go:112)	XCHGL	AX, AX
	0x00d7 00215 (internal/vec/vec_avx512.go:59)	VADDPS	Z4, Z1, Z1
	0x00dd 00221 (<unknown line number>)	NOP
	0x00dd 00221 (internal/vec/vec_avx512.go:128)	XORL	BX, BX
	0x00df 00223 (internal/vec/vec_avx512.go:128)	NOP
	0x00e0 00224 (internal/l1/l1_amd64.go:241)	JMP	288
	0x00e2 00226 (internal/l1/l1_amd64.go:241)	MOVL	$2147483647, DX
	0x00e7 00231 (internal/l1/l1_amd64.go:236)	JMP	288
	0x00e9 00233 (internal/l1/l1_amd64.go:245)	ADDQ	$-16, CX
	0x00ed 00237 (internal/l1/l1_amd64.go:245)	MOVQ	CX, SI
	0x00f0 00240 (internal/l1/l1_amd64.go:245)	NEGQ	SI
	0x00f3 00243 (internal/l1/l1_amd64.go:245)	SARQ	$63, SI
	0x00f7 00247 (internal/l1/l1_amd64.go:245)	ANDL	$64, SI
	0x00fa 00250 (/Volumes/External HDSIMD/archsimd/other_gen_amd64.go:211)	VMOVD	DX, X4
	0x00fe 00254 (/Volumes/External HDSIMD/archsimd/other_gen_amd64.go:211)	VPBROADCASTD	X4, Z4
	0x0104 00260 (internal/vec/vec_avx512.go:154)	VPANDD	(AX), Z4, Z4
	0x010a 00266 (internal/l1/l1_amd64.go:244)	XCHGL	AX, AX
	0x010b 00267 (internal/l1/l1_amd64.go:244)	XCHGL	AX, AX
	0x010c 00268 (internal/l1/l1_amd64.go:245)	ADDQ	SI, AX
	0x010f 00271 (internal/vec/vec_avx512.go:112)	XCHGL	AX, AX
	0x0110 00272 (internal/vec/vec_avx512.go:112)	XCHGL	AX, AX
	0x0111 00273 (internal/vec/vec_avx512.go:59)	VADDPS	Z4, Z2, Z2
	0x0117 00279 (internal/vec/vec_avx512.go:128)	XCHGL	AX, AX
	0x0118 00280 (internal/l1/l1_amd64.go:245)	ADDQ	$-16, BX
	0x011c 00284 (internal/l1/l1_amd64.go:245)	NOP
	0x0120 00288 (internal/l1/l1_amd64.go:243)	CMPQ	BX, $16
	0x0124 00292 (internal/l1/l1_amd64.go:243)	JGE	233
	0x0126 00294 (internal/l1/l1_amd64.go:247)	TESTQ	BX, BX
	0x0129 00297 (internal/l1/l1_amd64.go:247)	JEQ	360
	0x012b 00299 (/Volumes/External HDSIMD/archsimd/slice_gen_amd64.go:654)	LEAQ	-16(BX), CX
	0x012f 00303 (/Volumes/External HDSIMD/archsimd/slice_gen_amd64.go:654)	NEGQ	CX
	0x0132 00306 (/Volumes/External HDSIMD/archsimd/slice_gen_amd64.go:654)	MOVL	$-1, BX
	0x0137 00311 (/Volumes/External HDSIMD/archsimd/slice_gen_amd64.go:654)	SHRW	CX, BX
	0x013a 00314 (/Volumes/External HDSIMD/archsimd/slice_gen_amd64.go:654)	KMOVW	BX, K1
	0x013e 00318 (/Volumes/External HDSIMD/archsimd/slice_gen_amd64.go:654)	VPMOVM2D	K1, Z4
	0x0144 00324 (/Volumes/External HDSIMD/archsimd/slice_gen_amd64.go:655)	VMOVDQU64	(AX), Z5
	0x014a 00330 (/Volumes/External HDSIMD/archsimd/other_gen_amd64.go:211)	VMOVD	DX, X6
	0x014e 00334 (/Volumes/External HDSIMD/archsimd/other_gen_amd64.go:211)	VPBROADCASTD	X6, Z6
	0x0154 00340 (internal/vec/vec_avx512.go:154)	VPTERNLOGD	$128, Z6, Z4, Z5
	0x015b 00347 (internal/l1/l1_amd64.go:248)	XCHGL	AX, AX
	0x015c 00348 (internal/l1/l1_amd64.go:248)	XCHGL	AX, AX
	0x015d 00349 (internal/l1/l1_amd64.go:248)	XCHGL	AX, AX
	0x015e 00350 (internal/vec/vec_avx512.go:44)	XCHGL	AX, AX
	0x015f 00351 (internal/vec/vec_avx512.go:112)	XCHGL	AX, AX
	0x0160 00352 (internal/vec/vec_avx512.go:112)	XCHGL	AX, AX
	0x0161 00353 (internal/vec/vec_avx512.go:59)	VADDPS	Z5, Z2, Z2
	0x0167 00359 (internal/vec/vec_avx512.go:128)	XCHGL	AX, AX
	0x0168 00360 (internal/vec/vec_avx512.go:59)	VADDPS	Z3, Z2, Z2
	0x016e 00366 (internal/vec/vec_avx512.go:59)	VADDPS	Z1, Z0, Z1
	0x0174 00372 (internal/vec/vec_avx512.go:59)	VADDPS	Z1, Z2, Z1
	0x017a 00378 (internal/vec/vec_avx512.go:166)	VEXTRACTF64X4	$0, Z1, Y2
	0x0181 00385 (internal/vec/vec_avx512.go:166)	VEXTRACTF64X4	$1, Z1, Y1
	0x0188 00392 (internal/vec/vec_avx512.go:166)	VADDPS	Y1, Y2, Y1
	0x018c 00396 (internal/vec/vec_avx512.go:167)	VEXTRACTF128	$0, Y1, X2
	0x0192 00402 (internal/vec/vec_avx512.go:167)	VEXTRACTF128	$1, Y1, X1
	0x0198 00408 (internal/vec/vec_avx512.go:167)	VADDPS	X1, X2, X1
	0x019c 00412 (<unknown line number>)	NOP
	0x019c 00412 (<unknown line number>)	NOP
	0x019c 00412 (<unknown line number>)	NOP
	0x019c 00412 (<unknown line number>)	NOP
	0x019c 00412 (internal/vec/vec_avx512.go:169)	VMOVDQU	X1, github.com/scttfrdmn/keel/internal/vec.a(SP)
	0x01a1 00417 (internal/vec.a+8(SP), X0
	0x01a7 00423 (internal/vec.a(SP), X0
	0x01ac 00428 (internal/vec/vec_avx512.go:170)	MOVSS	X0, github.com/scttfrdmn/keel/internal/vec.a(SP)
	0x01b1 00433 (internal/vec.a+12(SP), X1
	0x01b7 00439 (internal/vec.a+4(SP), X1
	0x01bd 00445 (internal/vec/vec_avx512.go:171)	MOVSS	X1, github.com/scttfrdmn/keel/internal/vec.a+4(SP)
	0x01c3 00451 (internal/vec/vec_avx512.go:172)	ADDSS	X1, X0
	0x01c7 00455 (internal/l1/l1_amd64.go:250)	ADDQ	$16, SP
	0x01cb 00459 (internal/l1/l1_amd64.go:250)	POPQ	BP
	0x01cc 00460 (internal/l1/l1_amd64.go:250)	RET
	0x0000 55 48 89 e5 48 83 ec 10 48 89 44 24 20 90 62 f1  UH..H...H.D$ .b.
	0x0010 f5 48 ef c9 62 f1 fe 48 7f c8 62 f1 fe 48 7f c2  .H..b..H..b..H..
	0x0020 62 f1 fe 48 7f d3 eb 5e ba ff ff ff 7f c5 f9 6e  b..H...^.......n
	0x0030 e2 62 f2 7d 48 58 e4 62 f1 5d 48 db 28 62 f1 5d  .b.}HX.b.]H.(b.]
	0x0040 48 db 70 01 62 f1 5d 48 db 78 02 62 f1 5d 48 db  H.p.b.]H.x.b.]H.
	0x0050 60 03 90 90 90 90 90 90 90 90 90 90 62 f1 6c 48  `...........b.lH
	0x0060 58 d5 90 62 f1 64 48 58 de 90 62 f1 7c 48 58 c7  X..b.dHX..b.|HX.
	0x0070 90 62 f1 74 48 58 cc 90 48 83 c1 c0 48 05 00 01  .b.tHX..H...H...
	0x0080 00 00 48 83 c3 c0 48 83 fb 40 7f 9c 75 54 ba ff  ..H...H..@..uT..
	0x0090 ff ff 7f c5 f9 6e e2 62 f2 7d 48 58 e4 62 f1 5d  .....n.b.}HX.b.]
	0x00a0 48 db 28 62 f1 5d 48 db 70 01 62 f1 5d 48 db 78  H.(b.]H.p.b.]H.x
	0x00b0 02 62 f1 5d 48 db 60 03 90 90 90 90 90 90 90 90  .b.]H.`.........
	0x00c0 90 90 62 f1 6c 48 58 d5 90 62 f1 64 48 58 de 90  ..b.lHX..b.dHX..
	0x00d0 62 f1 7c 48 58 c7 90 62 f1 74 48 58 cc 31 db 90  b.|HX..b.tHX.1..
	0x00e0 eb 3e ba ff ff ff 7f eb 37 48 83 c1 f0 48 89 ce  .>......7H...H..
	0x00f0 48 f7 de 48 c1 fe 3f 83 e6 40 c5 f9 6e e2 62 f2  H..H..?..@..n.b.
	0x0100 7d 48 58 e4 62 f1 5d 48 db 20 90 90 48 01 f0 90  }HX.b.]H. ..H...
	0x0110 90 62 f1 6c 48 58 d4 90 48 83 c3 f0 0f 1f 40 00  .b.lHX..H.....@.
	0x0120 48 83 fb 10 7d c3 48 85 db 74 3d 48 8d 4b f0 48  H...}.H..t=H.K.H
	0x0130 f7 d9 bb ff ff ff ff 66 d3 eb c5 f8 92 cb 62 f2  .......f......b.
	0x0140 7e 48 38 e1 62 f1 fe 48 6f 28 c5 f9 6e f2 62 f2  ~H8.b..Ho(..n.b.
	0x0150 7d 48 58 f6 62 f3 5d 48 25 ee 80 90 90 90 90 90  }HX.b.]H%.......
	0x0160 90 62 f1 6c 48 58 d5 90 62 f1 6c 48 58 d3 62 f1  .b.lHX..b.lHX.b.
	0x0170 7c 48 58 c9 62 f1 6c 48 58 c9 62 f3 fd 48 1b ca  |HX.b.lHX.b..H..
	0x0180 00 62 f3 fd 48 1b c9 01 c5 ec 58 c9 c4 e3 7d 19  .b..H.....X...}.
	0x0190 ca 00 c4 e3 7d 19 c9 01 c5 e8 58 c9 c5 fa 7f 0c  ....}.....X.....
	0x01a0 24 f3 0f 10 44 24 08 f3 0f 58 04 24 f3 0f 11 04  $...D$...X.$....
	0x01b0 24 f3 0f 10 4c 24 0c f3 0f 58 4c 24 04 f3 0f 11  $...L$...XL$....
	0x01c0 4c 24 04 f3 0f 58 c1 48 83 c4 10 5d c3           L$...X.H...].
