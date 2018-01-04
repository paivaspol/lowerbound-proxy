#!/usr/bin/env bash
proxy_dir=$1
proxy_port=$2
webserver_port=$3
prefetch_urls=$4
request_order=$5

if [ "$#" -ne 5 ]; then
	echo "Usage: ./run_proxy.sh [proxy_dir] [proxy_port] [webserver_port] [prefetch_urls] [request_order]"
	exit 1
fi

echo "Starting proxy.go..."
go run ${proxy_dir}/proxy/proxy.go -port=${proxy_port} -prefetch-urls=${prefetch_urls} -request-order=${request_order} &> proxy.out &

echo "Starting prefetchwebserver.go..."
go run ${proxy_dir}/prefetch_webserver/prefetchwebserver.go -port=${webserver_port} -prefetch-urls=${prefetch_urls} &> prefetchserver.out &
