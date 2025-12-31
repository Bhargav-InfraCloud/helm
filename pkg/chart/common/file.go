/*
Copyright The Helm Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package common

import (
	"time"
)

// File represents a file as a name/value pair.
//
// By convention, name is a relative path within the scope of the chart's
// base directory.
type File struct {
	// Name is the path-like name of the template.
	Name string `json:"name"`
	// Data is the template as byte data.
	Data []byte `json:"data"`
	// ModTime is the file's mod-time
	ModTime time.Time `json:"modtime,omitzero"`
}

// // DeepCopy creates a deep copy of the File object.
// func (f *File) DeepCopy() *File {
// 	// If the file is nil, return nil.
// 	if f == nil {
// 		return nil
// 	}

// 	// Create a shallow copy of the file.
// 	//
// 	// This is enough to copy the following fields:
// 	// - Name (string)
// 	// - ModTime (time.Time)
// 	//
// 	// For the other field, i.e., Data, a deep copy is needed.
// 	newFile := *f

// 	// Deep copy the Data (byte slice).
// 	newFile.Data = bytes.Clone(f.Data)

// 	return &newFile
// }
