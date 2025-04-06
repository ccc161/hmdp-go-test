package models

import "encoding/json"

type Result struct {
	Success bool         `json:"success"`
	Data    DataAsString `json:"data"`
}

type DataAsString string

func (d *DataAsString) String() string {
	return string(*d)
}

func (d *DataAsString) UnmarshalJSON(data []byte) error {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	switch x := v.(type) {
	case string:
		*d = DataAsString(x)
	case nil:
		*d = DataAsString("null") // 处理null情况
	default:
		// 将其他类型编码为JSON字符串
		b, err := json.Marshal(x)
		if err != nil {
			return err
		}
		*d = DataAsString(b)
	}
	return nil
}
