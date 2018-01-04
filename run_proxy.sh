#!/usr/bin/env bash
proxy_port=$1
webserver_port=$2
prefetch_urls=$3
request_order=$4

if [ "$#" -ne 4 ]; then
	echo "Usage: ./run_proxy.sh [proxy_port] [webserver_port] [prefetch_urls] [request_order]"
fi

echo "Starting proxy.go..."
go run proxy/proxy.go -port=${proxy_port} -important-urls=${prefetch_urls} -request-order=${request_order} &> proxy.out &

echo "Starting prefetchwebserver.go..."
go run prefetch_webserver/prefetchwebserver.go -port=${webserver_port} -prefetch-urls=${prefetch_urls} &> prefetchserver.out &
