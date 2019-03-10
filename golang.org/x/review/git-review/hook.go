// Copyright 2014 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
)

var hookFile = filepath.FromSlash(".git/hooks/commit-msg")

func installHook() {
	filename := filepath.Join(repoRoot(), hookFile)
	_, err := os.Stat(filename)
	if err == nil {
		return
	}
	if !os.IsNotExist(err) {
		dief("error checking for hook file: %v", err)
	}
	verbosef("Presubmit hook to add Change-Id to commit messages is missing.")
	verbosef("Automatically creating it at %v.", filename)
	hookContent := []byte(commitMsgHook)
	if err := ioutil.WriteFile(filename, hookContent, 0700); err != nil {
		dief("error writing hook file: %v", err)
	}
}

func repoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		dief("could not get current directory: %v", err)
	}
	rootlen := 1
	if runtime.GOOS == "windows" {
		rootlen += len(filepath.VolumeName(dir))
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		if len(dir) == rootlen && dir[rootlen-1] == filepath.Separator {
			dief("git root not found. Rerun from within the Git tree.")
		}
		dir = filepath.Dir(dir)
	}
}

var commitMsgHook = `#!/bin/sh
# From Gerrit Code Review 2.2.1
#
# Part of Gerrit Code Review (http://code.google.com/p/gerrit/)
#
# Copyright (C) 2009 The Android Open Source Project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

CHANGE_ID_AFTER="Bug|Issue"
MSG="$1"

# Check for, and add if missing, a unique Change-Id
#
add_ChangeId() {
	clean_message=` + "`" + `sed -e '
		/^diff --git a\/.*/{
			s///
			q
		}
		/^Signed-off-by:/d
		/^#/d
	' "$MSG" | git stripspace` + "`" + `
	if test -z "$clean_message"
	then
		return
	fi

	if grep -i '^Change-Id:' "$MSG" >/dev/null
	then
		return
	fi

	id=` + "`" + `_gen_ChangeId` + "`" + `
	perl -e '
		$MSG = shift;
		$id = shift;
		$CHANGE_ID_AFTER = shift;

		undef $/;
		open(I, $MSG); $_ = <I>; close I;
		s|^diff --git a/.*||ms;
		s|^#.*$||mg;
		exit unless $_;

		@message = split /\n/;
		$haveFooter = 0;
		$startFooter = @message;
		for($line = @message - 1; $line >= 0; $line--) {
			$_ = $message[$line];

			if (/^[a-zA-Z0-9-]+:/ && !m,^[a-z0-9-]+://,) {
				$haveFooter++;
				next;
			}
			next if /^[ []/;
			$startFooter = $line if ($haveFooter && /^\r?$/);
			last;
		}

		@footer = @message[$startFooter+1..@message];
		@message = @message[0..$startFooter];
		push(@footer, "") unless @footer;

		for ($line = 0; $line < @footer; $line++) {
			$_ = $footer[$line];
			next if /^($CHANGE_ID_AFTER):/i;
			last;
		}
		splice(@footer, $line, 0, "Change-Id: I$id");

		$_ = join("\n", @message, @footer);
		open(O, ">$MSG"); print O; close O;
	' "$MSG" "$id" "$CHANGE_ID_AFTER"
}
_gen_ChangeIdInput() {
	echo "tree ` + "`" + `git write-tree` + "`" + `"
	if parent=` + "`" + `git rev-parse HEAD^0 2>/dev/null` + "`" + `
	then
		echo "parent $parent"
	fi
	echo "author ` + "`" + `git var GIT_AUTHOR_IDENT` + "`" + `"
	echo "committer ` + "`" + `git var GIT_COMMITTER_IDENT` + "`" + `"
	echo
	printf '%s' "$clean_message"
}
_gen_ChangeId() {
	_gen_ChangeIdInput |
	git hash-object -t commit --stdin
}


add_ChangeId
`
