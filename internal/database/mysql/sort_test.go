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

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArraySliceSort(t *testing.T) {
	a1 := [][2]int{
		{2, 4},
		{9, 11},
		{1, 3},
		{6, 6},
		{13, 13},
		{12, 12},
		{8, 9},
	}
	want := [][2]int{
		{1, 3},
		{2, 4},
		{6, 6},
		{8, 9},
		{9, 11},
		{12, 12},
		{13, 13},
	}
	sort.Sort(ArraySlice(a1))
	assert.Equal(t, want, a1)
}
