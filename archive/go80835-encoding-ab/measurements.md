# toolchain compile-binary sha256 (the two toolchains genuinely differ)
2f5c402ead4f6e365be1e4437261078f42aabc9858578a75a685b89916ea58fe  /home/scttfrdmn/vzup/go-fixed/pkg/tool/linux_amd64/compile
5a2eecfb7831296b0ac9ded5bfc0c6184bf668df4b7758b0b352576d14c4f1a4  /home/scttfrdmn/vzup/go-legacy/pkg/tool/linux_amd64/compile
# keel bench binary sha256 (differ only by embedded path/build id)
2242d52b4682133b965473db6a6e3f091e771f16e61c2cedb6562e0590e4ad1b  /home/scttfrdmn/vzup/kern-fixed.test
835f54ae623813b6862b4aded1d8bfb408d9eea163220a329458e40a66eedca3  /home/scttfrdmn/vzup/kern-legacy.test
# normalized disassembly sha256 (filename header stripped) — IDENTICAL
428d3575591d4ab55af993e8a0e58a70aa14bafca068a31b3282272bf32155fc  /tmp/n-fixed.txt
428d3575591d4ab55af993e8a0e58a70aa14bafca068a31b3282272bf32155fc  /tmp/n-legacy.txt
# legacy movups/movaps, whole binary
fixed 6416
legacy 6416
# vector-move census inside Kernel6x32|2x32|4x32 (fixed arm)
    177 vmovdqu64
     88 vfmadd213ps
     62 mov
     44 vmovss
     44 vbroadcastss
     36 movups
# the staged revert
 test/simd.go                                       | 32 +----------
 test/simd_critical.go                              | 66 ----------------------
 4 files changed, 1 insertion(+), 136 deletions(-)
