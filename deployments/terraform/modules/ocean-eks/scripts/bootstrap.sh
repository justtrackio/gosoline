#!/bin/bash
set -o xtrace
/etc/eks/bootstrap.sh ${default_label}

${nvme}