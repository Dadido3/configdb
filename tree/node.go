// Copyright (c) 2019 David Vogel
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package tree

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
)

// Node contains children that are either nodes, arrays, or values of the following types:
// - bool
// - string
// - number
//
// Valid child names must not contain periods (PathSeparator).
type Node map[string]interface{}

// CreatePath makes sure that a given path exists by creating nodes and overwriting existing values.
//
// The function will return the node the path points to.
func (n Node) CreatePath(path string) Node {
	elements := PathSplit(path)

	node := n
	for _, e := range elements {
		child, ok := node[e]
		if !ok {
			// Create child if it doesn't exist
			tempNode := Node{}
			node[e] = tempNode
			node = tempNode
		} else {
			tempNode, ok := child.(Node)
			if !ok {
				// Child is not a node, so overwrite it
				tempNode = Node{}
				node[e] = tempNode
			}
			node = tempNode
		}
	}

	return node
}

// Set creates all needed nodes and sets the element at the given path.
func (n Node) Set(path string, element interface{}) error {
	var newElement interface{}

	switch v := element.(type) {
	case Node, bool, string:
		newElement = v
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		result, err := NumberCreate(v)
		if err != nil {
			return err
		}
		newElement = result
	case []Node, []bool, []string, []Number: // TODO: Add more array types (Special case: []byte). Also array of arrays
		newElement = v
	// TODO: Handle any structure, and split it into its base types/arrays/nodes
	default:
		return ErrUnexpectedType{"", fmt.Sprintf("%T", v), ""}
	}

	pathElements := PathSplit(path)
	lastElement := pathElements[len(pathElements)-1]
	node := n
	if len(pathElements) > 1 {
		node = n.CreatePath(PathJoin(pathElements[:len(pathElements)-1]...))
	}
	node[lastElement] = newElement

	return nil
}

// GetOrError returns a node or value at the given path, or an error.
func (n Node) GetOrError(path string) (interface{}, error) {
	elements := PathSplit(path)

	inter := interface{}(n)
	for _, e := range elements {
		var ok bool
		node, ok := inter.(Node)
		if !ok {
			return nil, ErrPathInsideValue{path} // Path points inside a value
		}
		inter, ok = node[e]
		if !ok {
			return nil, ErrElementNotFound{path} // Element at path doesn't exist
		}
	}

	return inter, nil
}

// Get returns a node or value at the given path, or nil if it can't be found.
func (n Node) Get(path string) interface{} {
	result, err := n.GetOrError(path)
	if err != nil {
		return nil
	}

	return result
}

// TODO: Add GetStruct method
// TODO: Add GetArray* methods

// GetBoolOrError returns the bool at the given path, or an error if it doesn't exist.
func (n Node) GetBoolOrError(path string) (bool, error) {
	inter, err := n.GetOrError(path)
	if err != nil {
		return false, err
	}
	if v, ok := inter.(bool); ok {
		return v, nil
	}

	return false, ErrUnexpectedType{path, fmt.Sprintf("%T", inter), "bool"}
}

// GetBool returns the bool at the given path.
// In case of an error, the fallback is returned.
func (n Node) GetBool(path string, fallback bool) bool {
	result, err := n.GetBoolOrError(path)
	if err != nil {
		return fallback
	}

	return result
}

// GetStringOrError returns the string at the given path, or an error if it doesn't exist.
func (n Node) GetStringOrError(path string) (string, error) {
	inter, err := n.GetOrError(path)
	if err != nil {
		return "", err
	}
	if v, ok := inter.(string); ok {
		return v, nil
	}

	return "", ErrUnexpectedType{path, fmt.Sprintf("%T", inter), "string"}
}

// GetString returns the string at the given path.
// In case of an error, the fallback is returned.
func (n Node) GetString(path string, fallback string) string {
	result, err := n.GetStringOrError(path)
	if err != nil {
		return fallback
	}

	return result
}

// GetInt64OrError returns the integer at the given path, or an error if it doesn't exist.
func (n Node) GetInt64OrError(path string) (int64, error) {
	inter, err := n.GetOrError(path)
	if err != nil {
		return 0, err
	}
	if v, ok := inter.(Number); ok {
		return v.Int64()
	}

	return 0, ErrUnexpectedType{path, fmt.Sprintf("%T", inter), "number"}
}

// GetInt64 returns the integer at the given path.
// In case of an error, the fallback is returned.
func (n Node) GetInt64(path string, fallback int64) int64 {
	result, err := n.GetInt64OrError(path)
	if err != nil {
		return fallback
	}

	return result
}

// GetFloat64OrError returns the float at the given path, or an error if it doesn't exist.
func (n Node) GetFloat64OrError(path string) (float64, error) {
	inter, err := n.GetOrError(path)
	if err != nil {
		return 0, err
	}
	if v, ok := inter.(Number); ok {
		return v.Float64()
	}

	return 0, ErrUnexpectedType{path, fmt.Sprintf("%T", inter), "number"}
}

// GetFloat64 returns the float at the given path.
// In case of an error, the fallback is returned.
func (n Node) GetFloat64(path string, fallback float64) float64 {
	result, err := n.GetFloat64OrError(path)
	if err != nil {
		return fallback
	}

	return result
}

// Compare compares the current tree with the one in new and returns a list of paths for elements that were modified, added or removed.
//
// A change of the content/sub-content of an array is returned as change of the array itself.
func (n Node) Compare(new Node) (modified, added, removed []string) {
	return n.compare(new, "")
}

func (n Node) compare(new Node, prefix string) (modified, added, removed []string) {
	// Look for modified or removed elements
	for k, v := range n {
		vNew, foundNew := new[k]

		if foundNew {
			nodeA, aIsNode := v.(Node)
			nodeB, bIsNode := vNew.(Node)
			if aIsNode && bIsNode {
				// If both elements are nodes, check recursively.
				mod, add, rem := nodeA.compare(nodeB, prefix+k+PathSeparator) // Prefix is not really a path, as it can have a path separator at the end
				modified, added, removed = append(modified, mod...), append(added, add...), append(removed, rem...)
			} else if aIsNode {
				// If only a is a node, it got overwritten by a value
				modified = append(modified, prefix+k)
				_, _, rem := nodeA.compare(Node{}, prefix+k+PathSeparator)
				removed = append(removed, rem...)
			} else if bIsNode {
				// If only b is a node, it replaced a value
				modified = append(modified, prefix+k)
				_, add, _ := Node{}.compare(nodeB, prefix+k+PathSeparator)
				added = append(added, add...)
			} else if !cmp.Equal(v, vNew) {
				// If the two values are not equal
				modified = append(modified, prefix+k)
			}
			continue
		}

		// Not found, add to removed list
		removed = append(removed, prefix+k)
		if nodeA, ok := v.(Node); ok {
			_, _, rem := nodeA.compare(Node{}, prefix+k+PathSeparator)
			removed = append(removed, rem...)
		}
	}

	// Look for added elements
	for k, vNew := range new {
		_, found := n[k]

		if !found {
			added = append(added, prefix+k)
			if nodeB, ok := vNew.(Node); ok {
				_, add, _ := Node{}.compare(nodeB, prefix+k+PathSeparator)
				added = append(added, add...)
			}
		}
	}

	return
}

// Merge merges this tree with the new one.
//
// The following rules apply:
// - If both elements are nodes, their children are merged
// - Otherwise, the element of the new tree is written
// - If there is some element in the old, but not in the new tree, the old one is kept
// - If there is some element in the new, but not in the old tree, the new one is written
//
// Arrays will not be merged, but new ones will overwrite old ones.
func (n Node) Merge(new Node) {
	for k, vNew := range new {
		v, found := n[k]

		if found {
			nodeA, aIsNode := v.(Node)
			nodeB, bIsNode := vNew.(Node)
			if aIsNode && bIsNode {
				// If both elements are nodes, merge recursively.
				nodeA.Merge(nodeB)
			} else {
				// If only one or none of the elements is a node, replace the old with the new one
				n[k] = vNew
			}
			continue
		}

		// Element not found in old tree
		n[k] = vNew
	}
}

// Check returns an error when a tree contains any malformed or illegal elements
func (n Node) Check() error {
	var recursive func(v interface{}, path string) error
	recursive = func(v interface{}, path string) error {
		switch v := v.(type) {
		case Node:
			for k, child := range v {
				err := recursive(child, PathJoin(path, k))
				if err != nil {
					return err
				}
			}
		case []Node: // TODO: Array values, arrays and other things (Use reflect package)
			for i, child := range v {
				err := recursive(child, PathJoin(path, fmt.Sprint(i))) // Pseudo path for array elements, not really a valid path
				if err != nil {
					return err
				}
			}
		case bool, string, Number:
		default:
			return ErrUnexpectedType{path, fmt.Sprintf("%T", v), ""}
		}

		return nil
	}

	return recursive(n, "")
}
