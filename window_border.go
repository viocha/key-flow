package main

import (
	"errors"
	"strconv"
	"strings"
)

func parseWindowBorderColor(value string) (uint32, error) {
	value = strings.TrimSpace(value)
	if len(value) != 7 || value[0] != '#' {
		return 0, errors.New("置顶边框颜色须为 #RRGGBB")
	}
	rgb, err := strconv.ParseUint(value[1:], 16, 24)
	if err != nil {
		return 0, errors.New("置顶边框颜色须为 #RRGGBB")
	}
	red := uint32((rgb >> 16) & 0xff)
	green := uint32((rgb >> 8) & 0xff)
	blue := uint32(rgb & 0xff)
	// Win32 COLORREF 使用 0x00BBGGRR。
	return red | green<<8 | blue<<16, nil
}
