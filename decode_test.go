package bru

import (
	"fmt"
	"testing"
)

// TODO: this could be improved
func TestDecodingMetaSimple(t *testing.T) {
	simpleFile := `meta {
	url: https://toto.com
}`
	read, err := Read([]byte(simpleFile))
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(read[0])
}

func TestDecodingMetaMulti(t *testing.T) {
	simpleFile := `meta {
	url: https://toto.com,
 toto: abcd.com
}`
	read, err := Read([]byte(simpleFile))
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(read[0])
}

func TestDecodingVarMulti(t *testing.T) {
	simpleFile := `vars:secret [
  access_key,
  access_secret,
  ~transactionId
]`
	read, err := Read([]byte(simpleFile))
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(read[0])
}

func TestDecodingMultiBlocks(t *testing.T) {
	simpleFile := `body {
  {
    "hello": "world"
  }
}

tests {
  expect(res.status).to.equal(200);
}`
	read, err := Read([]byte(simpleFile))
	if err != nil {
		t.Fatal(err.Error())
	}
	for _, b := range read {
		//t.Log(b)
		fmt.Println("------------")
		fmt.Println(b.(*TextBlock).Content)
		fmt.Println("------------")
	}
}

func TestDecodingMultiBlocksMultiType(t *testing.T) {
	simpleFile := `body {
  {
    "hello": "world"
  }
}

tests {
  expect(res.status).to.equal(200);
}

vars:secret [
  access_key,
  access_secret,
  ~transactionId
]

meta {
	url: https://toto.com,
 toto: abcd.com
}`
	read, err := Read([]byte(simpleFile))
	if err != nil {
		t.Fatal(err.Error())
	}
	for _, b := range read {
		//t.Log(b)
		fmt.Println("------------")
		fmt.Println(b)
		fmt.Println("------------")
	}
}
