#!/bin/sh

# 1. global
ROOTPATH=`pwd`
ESPATH=$ROOTPATH/es
SRCPATH=$ROOTPATH/src
TESTPATH=$ROOTPATH/test

# 2. elasticsearch
cd $ESPATH
sh es.sh
sleep 30

# 3. source
cd $SRCPATH
sh src.sh

# 4. set server backends
sleep 20
cd $TESTPATH
sh test.sh

