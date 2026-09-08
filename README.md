# frame

Generic utilities for scanning, checksumming and pooling binary frames.

## Scanning Performance

`frame` is quite efficient. The pool uses size buckets which are pre-warmed with 
3 frames each; real-life efficiency will depend on the level of concurrency and 
how well the payloads fit in the pool buckets. Buckets are derived from caller 
provided maximum "poolable" payload size.

The following sample benchmarks ran with:

- 16B headers and 128B payloads (144B total data)
- 1024 frames for the scanner tests

_Note that `frame` scanning is not absolutely zero-allocation as these benchmarks
seem to suggest: the pool bucket warmup was done outside of the benchmarking to
isolate per record allocation metrics._

```
goos: linux
goarch: amd64
pkg: github.com/johnknl/frame
cpu: AMD Ryzen 9 5950X 16-Core Processor
BenchmarkCRC32C_Sum-32          13765294               104.4 ns/op      9957.88 MB/s          16 B/op          1 allocs/op
BenchmarkCRC32C_Validate-32     16505458               122.2 ns/op      8510.85 MB/s          16 B/op          1 allocs/op
BenchmarkReader_Read/full_payload-32            37128774                33.93 ns/op     30647.99 MB/s          0 B/op          0 allocs/op
BenchmarkReader_Read/header_only_limit_zero-32          53720100                22.44 ns/op      713.12 MB/s           0 B/op          0 allocs/op
BenchmarkScanner_Scan/default-32                           39490             32282 ns/op        4356.20 MB/s           144.0 bytes/record         31720864 records/s           0 B/op          0 allocs/op
BenchmarkScanner_Scan/with_validation-32                    8335            131188 ns/op        1071.94 MB/s           144.0 bytes/record          7805635 records/s       16409 B/op       1024 allocs/op
BenchmarkScanner_Scan/at_offset-32                         49960             24826 ns/op        4248.34 MB/s           144.0 bytes/record         30935500 records/s           0 B/op          0 allocs/op
PASS
```

### Scanning Large Streams

`frame` is intended for scanning large streams with frames addressed by logical offset (index).

Seeking by index requires reading each consecutive frame header, hence seeking to the nearest offset
before the header at the desired index, can have a large impact on performance when dealing with large
streams.

It does not implement any logical to physical offset index, callers are expected to maintain their
own (sparse) index at append or load time. This index can then be used to advance the scanner closer 
to the position of the desired index before seeking by logical offset, eg:

```go
byteOffset := sparseIndex.Floor(logicalOffset)
scanner.Seek(byteOffset, io.SeekStart)
scanner.SeekIndex(logicalOffset)

for scanner.Scan() {
  frame := scanner.Frame()
  ...
}
```

Example of load-time scanning with integrity check:

```go
pool := frame.NewPool[exampleHeader](256, 1<<20)
reader := frame.NewReader(stream, pool, frame.HeadersOnly)
scanner := frame.NewScanner(
	reader,
	frame.WithScannerValidator[exampleHeader](frame.NewCRC32C[exampleHeader]()),
)

defer scanner.Close()

for scanner.Scan() {
	sparseIndex.MaybeSet(scanner.Index(), scanner.Offset())
}
```


## Header

A concrete Header is the way to define a frame format. It is expected to be a fixed-size array of bytes, 
and implement the Header interface which constrains the size to 8, 16, 32, 64, or 128 bytes.

```go
type exampleHeader [16]byte

func (h exampleHeader) PayloadLength() uint32 {
	return binary.BigEndian.Uint32(h[0:4])
}

func (h exampleHeader) Index() uint32 {
	return binary.BigEndian.Uint32(h[4:8])
}

func (h exampleHeader) Checksum() uint32 {
	return binary.BigEndian.Uint32(h[12:16])
}

func (h exampleHeader) ChecksumBytes() []byte {
	return h[:12]
}

func (h *exampleHeader) setChecksum(sum uint32) {
	binary.BigEndian.PutUint32(h[12:16], sum)
}
```

## Examples

<!-- EXAMPLE:ExampleNewPool:start -->
## NewPool

Payload slices are reset when a frame is returned, allowing
callers to reuse allocations across reads and writes.

Go reference: [NewPool](https://pkg.go.dev/github.com/johnknl/frame#NewPool).

The following example shows borrowing and returning frames through Pool.

```go
pool := frame.NewPool[exampleHeader](64, 1024)
payload := []byte("ok")
var h exampleHeader
binary.BigEndian.PutUint32(h[0:4], uint32(len(payload)))
binary.BigEndian.PutUint32(h[4:8], 0)

f := pool.Get(uint32(len(payload)))
f.Set(h, payload)
fmt.Println(len(f.Payload), cap(f.Payload) >= 2)
f.Return()

g := pool.Get(0)
fmt.Println(len(g.Payload))
g.Return()

// Output:
// 2 true
// 0
```
<!-- EXAMPLE:ExampleNewPool:end -->

<!-- EXAMPLE:ExampleCRC32C_Validate:start -->
## CRC32C

### Validate

A frame is expected to have a checksum field in its header, which can be
validated against the header and payload. This is a simple integrity check to
detect corruption of the frame header and/or payload.

Go reference: [CRC32C.Validate](https://pkg.go.dev/github.com/johnknl/frame#CRC32C.Validate).

The following example shows how to stamp and then validate a frame checksum.

```go
payload := []byte("hello")
var h exampleHeader
binary.BigEndian.PutUint32(h[0:4], uint32(len(payload)))
binary.BigEndian.PutUint32(h[4:8], 1)
f := &frame.Frame[exampleHeader]{
	Header:  h,
	Payload: payload,
}

crc := frame.NewCRC32C[exampleHeader]()
h.setChecksum(crc.Sum(f))
f.Header = h

fmt.Println(crc.Validate(f) == nil)

f.Payload[0] = 'H'
fmt.Println(errors.Is(crc.Validate(f), frame.ErrInvalidChecksum))

// Output:
// true
// true
```
<!-- EXAMPLE:ExampleCRC32C_Validate:end -->

<!-- EXAMPLE:ExampleScanner:start -->
## Scanner

The frame scanner is a convenient way to read frames from a stream,
such as a file or a buffer. To enable scanning, the header must implement
`Index() uint32`, in addition to the `Header` interface.

Go reference: [Scanner](https://pkg.go.dev/github.com/johnknl/frame#Scanner).

The following example shows sequential scanning with checksum validation enabled.

```go
encode := func(index uint32, payload []byte) []byte {
	var h exampleHeader
	binary.BigEndian.PutUint32(h[0:4], uint32(len(payload)))
	binary.BigEndian.PutUint32(h[4:8], index)

	f := &frame.Frame[exampleHeader]{Header: h, Payload: payload}
	crc := frame.NewCRC32C[exampleHeader]()
	h.setChecksum(crc.Sum(f))

	out := make([]byte, 0, exampleHeaderSize+len(payload))
	out = append(out, h[:]...)
	out = append(out, payload...)

	return out
}

raw := make([]byte, 0, 2*exampleHeaderSize+3)
raw = append(raw, encode(0, []byte("a"))...)
raw = append(raw, encode(1, []byte("bc"))...)

stream := bytes.NewReader(raw)
pool := frame.NewPool[exampleHeader](16, 256)
reader := frame.NewReader(stream, pool, frame.MaxPayloadSize)

scanner := frame.NewScanner(
	reader,
	frame.WithScannerValidator[exampleHeader](frame.NewCRC32C[exampleHeader]()),
)
defer scanner.Close()

count := 0
for scanner.Scan() {
	f := scanner.Frame()
	fmt.Printf("%d:%s\n", f.Header.Index(), string(f.Payload))
	count++
}

fmt.Println("err", scanner.Err() == nil, "count", count)

// Output:
// 0:a
// 1:bc
// err true count 2
```
<!-- EXAMPLE:ExampleScanner:end -->

<!-- EXAMPLE:ExampleNewScanner_withOptions:start -->
## NewScanner

### WithOptions

This pattern is useful when resuming from a known position in an append-only file,
such as after a checkpoint.

Go reference: [NewScanner](https://pkg.go.dev/github.com/johnknl/frame#NewScanner).

The following example shows scanner configuration via functional options.

```go
encode := func(index uint32, payload []byte) []byte {
	var h exampleHeader
	binary.BigEndian.PutUint32(h[0:4], uint32(len(payload)))
	binary.BigEndian.PutUint32(h[4:8], index)

	f := &frame.Frame[exampleHeader]{Header: h, Payload: payload}
	crc := frame.NewCRC32C[exampleHeader]()
	h.setChecksum(crc.Sum(f))

	out := make([]byte, 0, exampleHeaderSize+len(payload))
	out = append(out, h[:]...)
	out = append(out, payload...)

	return out
}

raw := make([]byte, 0, 3*exampleHeaderSize+6)
raw = append(raw, encode(10, []byte("aa"))...)
raw = append(raw, encode(11, []byte("bb"))...)
raw = append(raw, encode(12, []byte("cc"))...)

stream := bytes.NewReader(raw)
pool := frame.NewPool[exampleHeader](16, 256)
reader := frame.NewReader(stream, pool, frame.MaxPayloadSize)

startOffset := int64(exampleHeaderSize + 2)
scanner := frame.NewScanner(
	reader,
	frame.WithScannerOffset[exampleHeader](startOffset),
	frame.WithScannerIndex[exampleHeader](11),
	frame.WithScannerValidator[exampleHeader](frame.NewCRC32C[exampleHeader]()),
)
defer scanner.Close()

for scanner.Scan() {
	f := scanner.Frame()
	fmt.Printf("%d:%s\n", f.Header.Index(), string(f.Payload))
}

fmt.Println(scanner.Err() == nil)

// Output:
// 11:bb
// 12:cc
// true
```
<!-- EXAMPLE:ExampleNewScanner_withOptions:end -->

<!-- EXAMPLE:ExampleScanner_Close:start -->
### Close

Close returns any currently borrowed frame to its pool immediately instead of waiting
for another Scan call.

Go reference: [Scanner.Close](https://pkg.go.dev/github.com/johnknl/frame#Scanner.Close).

The following example shows explicit scanner cleanup for early-exit callers.

```go
raw := encodeRaw(0, []byte("x"))
stream := bytes.NewReader(raw)
pool := frame.NewPool[exampleHeader](16, 256)
reader := frame.NewReader(stream, pool, frame.HeadersOnly)

scanner := frame.NewScanner(reader)
if scanner.Scan() {
	fmt.Println(len(scanner.Frame().Payload))
}

scanner.Close()
fmt.Println(scanner.Frame() == nil)

// Output:
// 0
// true
```
<!-- EXAMPLE:ExampleScanner_Close:end -->

## License

MIT
