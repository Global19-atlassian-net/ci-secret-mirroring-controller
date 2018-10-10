package controller

import (
	"errors"
	"fmt"
	"strings"
)

// Configuration defines the action for the secret mirror
type Configuration struct {
	// Secrets holds mirroring configurations.
	Secrets []MirrorConfig `json:"secrets"`
}

// MirrorConfig defines a mirror mapping
type MirrorConfig struct {
	// From is the source of mirrored secret data
	From SecretLocation `json:"from"`

	// To is the destination of mirrored secret data
	To SecretLocation `json:"to"`
}

func (c *MirrorConfig) validate(parent string) []string {
	var messages []string
	for _, msg := range c.From.validate(fmt.Sprintf("%s.from", parent)) {
		messages = append(messages, msg)
	}
	for _, msg := range c.To.validate(fmt.Sprintf("%s.to", parent)) {
		messages = append(messages, msg)
	}
	return messages
}

func (c *MirrorConfig) String() string {
	return fmt.Sprintf("(%s -> %s)", c.From.String(), c.To.String())
}

// SecretLocation unambiguously identifies a secret on the cluster
type SecretLocation struct {
	// Namespace identifies the namespace for this secret
	Namespace string `json:"namespace"`

	// Name identifies the secret within the namespace
	Name string `json:"name"`
}

func (l *SecretLocation) validate(parent string) []string {
	var messages []string
	if len(l.Namespace) == 0 {
		messages = append(messages, fmt.Sprintf("%s.namespace: must not be empty", parent))
	}
	if len(l.Name) == 0 {
		messages = append(messages, fmt.Sprintf("%s.name: must not be empty", parent))
	}
	return messages
}

func (l *SecretLocation) String() string {
	return fmt.Sprintf("%s/%s", l.Namespace, l.Name)
}

func (l *SecretLocation) Equals(other SecretLocation) bool {
	return l.Namespace == other.Namespace && l.Name == other.Name
}

// Validate ensures that the configuration is valid
func (c *Configuration) Validate() error {
	if len(c.Secrets) == 0 {
		return errors.New("secret mirroring mappings are required")
	}

	var messages []string
	nodes, edges := map[SecretLocation]bool{}, map[SecretLocation][]SecretLocation{}
	for i, mapping := range c.Secrets {
		nodes[mapping.From] = false
		nodes[mapping.To] = false
		if destinations, exists := edges[mapping.From]; !exists {
			edges[mapping.From] = []SecretLocation{mapping.To}
		} else {
			edges[mapping.From] = append(destinations, mapping.To)
		}
		messages = append(messages, mapping.validate(fmt.Sprintf("secrets[%d]", i))...)
	}

	// cycles will cause the controller to go haywire, so we forbid them
	for _, cycle := range findCycles(nodes, edges) {
		var cycleFormatted []string
		for _, node := range cycle {
			cycleFormatted = append(cycleFormatted, node.String())
		}
		messages = append(messages, fmt.Sprintf("mirroring mapping contains the cycle [%s], which is forbidden", strings.Join(cycleFormatted, " -> ")))
	}

	if len(messages) > 0 {
		return fmt.Errorf("invalid mirroring mapping: %s\n", strings.Join(messages, "\n"))
	}
	return nil
}

// findCycles runs a DFS from every node to find at most one cycle per root node
func findCycles(nodes map[SecretLocation]bool, edges map[SecretLocation][]SecretLocation) [][]SecretLocation {
	var cycles [][]SecretLocation
	for {
		// loop over the map, running DFS from nodes we
		// have never visited before until we have visited
		// all of the nodes in the graph. this way, we will
		// output each cycle only once
		for node := range nodes {
			if !nodes[node] {
				if cycle, exists := findCycle([]SecretLocation{node}, nodes, edges); exists {
					cycles = append(cycles, cycle)
				}
				continue
			}
		}
		break
	}
	return cycles
}

func findCycle(path []SecretLocation, nodes map[SecretLocation]bool, edges map[SecretLocation][]SecretLocation) ([]SecretLocation, bool) {
	last := path[len(path)-1]
	if children, exists := edges[last]; !exists {
		return nil, false
	} else {
		for _, child := range children {
			childPath := append(path, child)
			nodes[child] = true
			for _, parent := range path {
				if child.Equals(parent) {
					// we have a cycle
					return childPath, true
				}
			}
			if cycle, cycleExists := findCycle(childPath, nodes, edges); cycleExists {
				return cycle, cycleExists
			}
		}
	}
	return nil, false
}
