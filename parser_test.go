package bru

import "testing"

func TestMetaMultiple(t *testing.T) {
	simpleFile := `meta {
  url: https://toto.com,
  toto: toto.abcd
}
`
	err := checkValid([]byte(simpleFile), &scanner{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMetaMultipleDisabled(t *testing.T) {
	simpleFile := `meta {
  url: https://toto.com,
  ~toto: toto.abcd,
  abcd: toto
}
`
	err := checkValid([]byte(simpleFile), &scanner{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestInvalidBlock(t *testing.T) {
	simpleFile := `meta {
  url: https://toto.com,
  toto: toto.abcd

`
	err := checkValid([]byte(simpleFile), &scanner{})
	if err == nil {
		t.Fatal(err)
	}
	t.Log(err.Error())
}

func TestMetaEmpty(t *testing.T) {
	simpleFile := `meta {
}
`
	err := checkValid([]byte(simpleFile), &scanner{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMetaSingle(t *testing.T) {
	simpleFile := `meta {
url: https://toto.com
}
`
	err := checkValid([]byte(simpleFile), &scanner{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMetaWrong(t *testing.T) {
	simpleFile := `meta {
	dzqdqzdqzdqz
}
`
	err := checkValid([]byte(simpleFile), &scanner{})
	if err == nil {
		t.Fatal("should have failed")
	}
	t.Log(err.Error())
}

func TestMetaWrong2(t *testing.T) {
	simpleFile := `meta {
	dzqdqzdqzdqz: zdiqhidh,
qdzidihqhdzqi
}
`
	err := checkValid([]byte(simpleFile), &scanner{})
	if err == nil {
		t.Fatal("should have failed")
	}
	t.Log(err.Error())
}

func TestMetaEmptyValue(t *testing.T) {
	simpleFile := `meta {
	dzqdqzdqzdqz: 
}
`
	err := checkValid([]byte(simpleFile), &scanner{})
	if err == nil {
		t.Fatal("should have failed")
	}
	t.Log(err.Error())
}
func TestVarsWithType(t *testing.T) {
	simpleFile := `vars:secret [
  access_key,
  access_secret,
  ~transactionId
]
`
	err := checkValid([]byte(simpleFile), &scanner{})
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestTextBlocks(t *testing.T) {
	simpleFile := `body {
  {
    "hello": "world"
  }
}

tests {
  expect(res.status).to.equal(200);
}`

	err := checkValid([]byte(simpleFile), &scanner{})
	if err != nil {
		t.Fatal(err.Error())
	}
}
