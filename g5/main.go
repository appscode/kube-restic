package main

import (
	"fmt"
	"errors"
	"encoding/json"
)

type PrefixType int

const (
	Smart   PrefixType = iota
	PodName
	None
)

func (m PrefixType) MarshalJSON() ([]byte, error) {
	switch m {
	case Smart:
		return []byte(`"Smart"`), nil
	case PodName:
		return []byte(`"PodName"`), nil
	case None:
		return []byte(`"None"`), nil
	}
	return nil, errors.New("jsontypes.PrefixType: Invalid PrefixType")
}

func (m *PrefixType) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("jsontypes.PrefixType: UnmarshalJSON on nil pointer")
	}
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	switch s {
	case "Smart", "":
		*m = Smart
	case "PodName":
		*m = PodName
	case "None":
		*m = None
	default:
		return errors.New("jsontypes.PrefixType: Invalid PrefixType")
	}
	return nil
}

func main() {
	var a PrefixType
	b, err := json.Marshal(a)
	fmt.Println(string(b), err)

	var c PrefixType
	e2 := json.Unmarshal([]byte(`"None"`), &c)
	fmt.Println(c, e2)
}
