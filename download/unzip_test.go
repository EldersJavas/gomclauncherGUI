package download

import (
	"io/ioutil"
	"testing"
)

func TestUnzip(t *testing.T) {
	b, err := ioutil.ReadFile("1.15.2.json")
	if err != nil {
		t.Fatal(err)
	}
	l, err := Newlibraries(b, 5)
	if err != nil {
		t.Fatal(err)
	}
	err = l.Unzip("")
	if err != nil {
		t.Fatal(err)
	}
}
