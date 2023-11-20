/*
 * Copyright 2022-2026 The NeoHA Authors.
 *
 * See the AUTHORS file for a list of contributors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
*/

package mysql

// ArraySlice definition
type ArraySlice [][2]int

func (s ArraySlice) Len() int {
	return len(s)
}

func (s ArraySlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s ArraySlice) Less(i, j int) bool {
	if s[i][0] < s[j][0] {
		return true
	} else if s[i][0] > s[j][0] {
		return false
	} else if s[i][1] < s[j][1] {
		return true
	} else {
		return false
	}
}
