# filestore-upload-test
For testing uploading files into file storage (currently aostor (github.com/tgulacsi/aostor) and weed-fs () are implemented).

## Options
 * -debug - print debug messages?
 * -dump - dump request/response?
 * -parallel.read - how many parallel read goroutines should read back uploaded files
 * -parallel.write - how many parallel goroutines should upload files?
 * -request.compressable - should the request be compressable?
 * -request.gzip - use Accept: gzip ?
 * -request.num - number of requests
 * -request.same - request repetition odds
 * -request.size.init - request initial size
 * -request.size.max - request maximal size
 * -request.size.step - request size step
 * -aostor - AOSTOR server addres (host:port/realm)
 * -weed - WEED-FS master server address (host:port)

