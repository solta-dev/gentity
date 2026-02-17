package main

import (
	"fmt"
)

const (
	outer uint8 = iota
	initial
	tagString
	optionName
	optionValue
)

type tagParseResult map[string]string
type tagsParseResult map[string]tagParseResult

type tagParser struct {
	state          uint8
	start          int
	tagName        string
	optName        string
	curTag         tagParseResult
	outerDelimiter byte
	Text           string
	Result         tagsParseResult
}

func (tp *tagParser) parse() error {
	tp.Result = make(tagsParseResult)
	tp.state = outer
	for i := 0; i < len(tp.Text); i++ {
		var err error
		switch tp.state {
		case outer:
			tp.parseOuter(i)
		case initial:
			tp.parseInitial(i)
		case tagString:
			err = tp.parseTagString(i)
		case optionName:
			tp.parseOptionName(i)
		case optionValue:
			tp.parseOptionValue(i)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (tp *tagParser) parseOuter(i int) {
	tp.outerDelimiter = tp.Text[i]
	tp.state = initial
	tp.start = i + 1
}

func (tp *tagParser) parseInitial(i int) {
	switch tp.Text[i] {
	case byte(':'), tp.outerDelimiter:
		tp.tagName = tp.Text[tp.start:i]
		tp.state = tagString
		tp.curTag = make(tagParseResult)
		tp.Result[tp.tagName] = tp.curTag
	case byte(' '):
		tp.start = i + 1
	}
}

func (tp *tagParser) parseTagString(i int) error {
	switch tp.Text[i] {
	case byte('"'):
		tp.state = optionName
		tp.start = i + 1
		tp.optName = ""
	case tp.outerDelimiter:
	default:
		return fmt.Errorf("tag delimiter %c doesn't supported", tp.Text[i])
	}
	return nil
}

func (tp *tagParser) parseOptionName(i int) {
	switch tp.Text[i] {
	case byte('='):
		tp.optName = tp.Text[tp.start:i]
		tp.state = optionValue
		tp.start = i + 1
	case byte(' '), byte('"'):
		tp.optName = tp.Text[tp.start:i]
		tp.curTag[tp.optName] = ""
		if tp.Text[i] == byte('"') {
			tp.state = initial
		}
		tp.start = i + 1
	}
}

func (tp *tagParser) parseOptionValue(i int) {
	if tp.Text[i] == byte(' ') || tp.Text[i] == byte('"') {
		optValue := tp.Text[tp.start:i]
		tp.curTag[tp.optName] = optValue
		if tp.Text[i] == byte(' ') {
			tp.state = optionName
		} else if tp.Text[i] == byte('"') {
			tp.state = initial
		}
		tp.start = i + 1
	}
}
