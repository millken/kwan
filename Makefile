#!/bin/bash
install:
	gb build kwan

verbose:
	gb build -f kwan

clean:
	rm -rf bin/*
