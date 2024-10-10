#protoc  --go_out=M=github.com/adoggie/ccsuites/message:. ./quote_period.proto
 protoc --go_out=. *.proto
 protoc --python_out=. *.proto
 cp *.py ../cmd/QuoteAggrService/scripts/