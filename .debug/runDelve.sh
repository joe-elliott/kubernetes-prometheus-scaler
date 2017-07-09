#!/bin/bash 

dlv debug --listen=0.0.0.0:2345 --headless=true --backend=native --log=true -- -assessment-interval=5s -prometheus-url=http://prometheus:9090
