Benchmarks

```sh
BenchmarkMap-16                                      146           8168550 ns/op         8289055 B/op          1 all
BenchmarkUniq-16                                      10         103630461 ns/op        34581937 B/op         23 all
BenchmarkUniqThoas-16                                  4         283696922 ns/op        92759996 B/op    2000015 all
BenchmarkUniqStruct-16                                 6         177865631 ns/op        70916293 B/op         26 all
BenchmarkUniqThoasStruct-16                            3         374595307 ns/op        127689090 B/op   2000018 all
BenchmarkChunk-16                                    111          10232891 ns/op         9581239 B/op      10001 all
BenchmarkChunkReduce-16                               99          11395557 ns/op         9626752 B/op      10001 all
BenchmarkChunkThoas-16                                13         112813206 ns/op        49245217 B/op    1100025 all
BenchmarkDifferenceRandomStruct-16                 75243             13358 ns/op            7366 B/op         30 all
BenchmarkDifferenceThoasRandomStruct-16              470           2306334 ns/op          344273 B/op      20820 all
BenchmarkDifferenceRandom-16                       47986             24955 ns/op           13975 B/op         28 all
BenchmarkDifferenceThoasRandom-16                    868           1476350 ns/op          178580 B/op      20820 all
BenchmarkDifferenceStatic-16                      110392             10376 ns/op            3285 B/op         14 all
BenchmarkDifferenceThoasStatic-16                   1707            719145 ns/op           90498 B/op      10704 all
BenchmarkIntersect-16                              97228             13425 ns/op            5578 B/op         25 all
BenchmarkIntersectThoas-16                         27148             46202 ns/op           15720 B/op        517 all
```
