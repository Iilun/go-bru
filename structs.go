package bru

import "errors"

type DictionaryElement struct {
	Key   string
	Value string
}

type DictionaryBlock struct {
	Name    string
	Type    string
	Content []DictionaryElement
}
type TextBlock struct {
	Name    string
	Type    string
	Content string
}
type ArrayBlock struct {
	Name    string
	Type    string
	Content []string
}

func (t *DictionaryBlock) GetType() string {
	return t.Type
}

func (t *TextBlock) GetType() string {
	return t.Type
}

func (t *ArrayBlock) GetType() string {
	return t.Type
}

func (t *DictionaryBlock) GetName() string {
	return t.Name
}

func (t *TextBlock) GetName() string {
	return t.Name
}

func (t *ArrayBlock) GetName() string {
	return t.Name
}

func (t *DictionaryBlock) SetContent(content any) error {
	switch c := content.(type) {
	case []DictionaryElement:
		t.Content = c
		return nil
	}
	return errors.New("wrong type to set for dictionary")
}

func (t *TextBlock) SetContent(content any) error {
	switch c := content.(type) {
	case string:
		t.Content = c
		return nil
	}
	return errors.New("wrong type to set for dictionary")
}

func (t *ArrayBlock) SetContent(content any) error {
	switch c := content.(type) {
	case []string:
		t.Content = c
		return nil
	}
	return errors.New("wrong type to set for dictionary")
}

type ContentBlock interface {
	GetType() string
	GetName() string
	SetContent(content any) error
}
