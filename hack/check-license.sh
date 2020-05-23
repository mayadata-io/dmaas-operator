#!/bin/bash
# Copyright 2020 The MayaData Authors.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#     https://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

FILE_LIST=$(mktemp)
echo ">> checking license header"
for file in $(find . -type f -iname '*.go') ; do
	awk 'NR<=3' $file | grep -Eq "(Copyright|generated|GENERATED)" || echo $file >> $FILE_LIST
done

if [ $(cat $FILE_LIST |wc -l) -ne 0 ]; then
	echo "license header checking failed. Following files doesn't have license"
	cat $FILE_LIST
	rm $FILE_LIST
	exit 1
fi
rm $FILE_LIST

