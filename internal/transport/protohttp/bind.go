package protohttp

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/uniedit/server/internal/utils/response"
)

// BindQuery 将 URL query 参数绑定到 dst（一般是 proto 生成的请求结构体指针）。
func BindQuery(c *gin.Context, dst any) error {
	if c == nil {
		return fmt.Errorf("gin context is nil")
	}
	return bindValues(dst, c.Request.URL.Query())
}

// BindVars 将路由路径变量绑定到 dst（变量名来自 google.api.http 的 {var} 占位符）。
func BindVars(c *gin.Context, dst any, keys ...string) error {
	if c == nil {
		return fmt.Errorf("gin context is nil")
	}
	if len(keys) == 0 {
		return nil
	}
	values := make(map[string][]string, len(keys))
	for _, k := range keys {
		if k == "" {
			continue
		}
		if v := c.Param(k); v != "" {
			values[k] = []string{v}
		}
	}
	return bindValues(dst, values)
}

// BindBody 将 HTTP body 绑定到 dst 或 dst 的某个字段。
//
// - bodyField == ""：不绑定 body
// - bodyField == "*"：将整个 JSON body 绑定到 dst
// - bodyField == "field"：将 JSON body 绑定到 dst.field（仅支持顶层字段）
func BindBody(c *gin.Context, dst any, bodyField string) error {
	if c == nil {
		return fmt.Errorf("gin context is nil")
	}
	if dst == nil {
		return fmt.Errorf("dst is nil")
	}

	bodyField = strings.TrimSpace(bodyField)
	if bodyField == "" {
		return nil
	}
	if strings.Contains(bodyField, ".") {
		return fmt.Errorf("暂不支持 body 绑定到嵌套字段: %s", bodyField)
	}
	if bodyField == "*" {
		return c.ShouldBindJSON(dst)
	}

	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr || dstValue.IsNil() {
		return fmt.Errorf("dst 必须是非空指针")
	}
	dstElem := dstValue.Elem()
	if dstElem.Kind() != reflect.Struct {
		return fmt.Errorf("dst 必须指向 struct")
	}

	fieldValue, ok := findFieldByName(dstElem, bodyField)
	if !ok {
		return fmt.Errorf("找不到 body 字段 %q", bodyField)
	}

	if fieldValue.Kind() == reflect.Ptr {
		if fieldValue.IsNil() {
			fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
		}
		return c.ShouldBindJSON(fieldValue.Interface())
	}
	if fieldValue.Kind() == reflect.Struct {
		return c.ShouldBindJSON(fieldValue.Addr().Interface())
	}
	if fieldValue.CanAddr() {
		return c.ShouldBindJSON(fieldValue.Addr().Interface())
	}
	return fmt.Errorf("body 字段 %q 不可取地址", bodyField)
}

// AbortBindError 统一处理绑定失败错误。
func AbortBindError(c *gin.Context, err error) {
	if err != nil {
		_ = c.Error(err)
		response.BadRequest(c, err.Error())
		return
	}
	response.BadRequest(c, "bad request")
}

// AbortServerError 统一处理服务端错误（默认不向外暴露内部错误详情）。
func AbortServerError(c *gin.Context, err error) {
	if err != nil {
		_ = c.Error(err)
	}
	response.InternalError(c, "")
}

// WriteOK 写入 200 OK 响应。
func WriteOK(c *gin.Context, out any) {
	c.JSON(http.StatusOK, out)
}

func bindValues(dst any, values map[string][]string) error {
	if dst == nil {
		return fmt.Errorf("dst is nil")
	}

	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr || dstValue.IsNil() {
		return fmt.Errorf("dst 必须是非空指针")
	}
	dstElem := dstValue.Elem()
	if dstElem.Kind() != reflect.Struct {
		return fmt.Errorf("dst 必须指向 struct")
	}
	dstType := dstElem.Type()

	for i := 0; i < dstType.NumField(); i++ {
		sf := dstType.Field(i)
		if sf.PkgPath != "" { // 非导出字段
			continue
		}
		fv := dstElem.Field(i)
		if !fv.CanSet() {
			continue
		}

		keys := fieldKeys(sf)
		for _, key := range keys {
			raw, ok := values[key]
			if !ok || len(raw) == 0 {
				continue
			}
			if err := setValue(fv, raw); err != nil {
				return fmt.Errorf("绑定字段 %s 失败: %w", sf.Name, err)
			}
			break
		}
	}
	return nil
}

func fieldKeys(sf reflect.StructField) []string {
	seen := map[string]struct{}{}
	var keys []string

	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || s == "-" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		keys = append(keys, s)
	}

	if jsonTag := sf.Tag.Get("json"); jsonTag != "" {
		name, _, _ := strings.Cut(jsonTag, ",")
		add(name)
	}

	if pbTag := sf.Tag.Get("protobuf"); pbTag != "" {
		for _, part := range strings.Split(pbTag, ",") {
			if k, v, ok := strings.Cut(part, "="); ok {
				switch k {
				case "name", "json":
					add(v)
				}
			}
		}
	}

	add(sf.Name)
	add(strings.ToLower(sf.Name))
	return keys
}

func findFieldByName(structValue reflect.Value, name string) (reflect.Value, bool) {
	t := structValue.Type()
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" {
			continue
		}
		if sf.Name == name {
			return structValue.Field(i), true
		}
		for _, key := range fieldKeys(sf) {
			if key == name {
				return structValue.Field(i), true
			}
		}
	}
	return reflect.Value{}, false
}

func setValue(v reflect.Value, raw []string) error {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		return setValue(v.Elem(), raw)
	}

	switch v.Kind() {
	case reflect.Slice:
		elemType := v.Type().Elem()
		s := reflect.MakeSlice(v.Type(), 0, len(raw))
		for _, item := range raw {
			elem := reflect.New(elemType).Elem()
			if err := setValue(elem, []string{item}); err != nil {
				return err
			}
			s = reflect.Append(s, elem)
		}
		v.Set(s)
		return nil
	case reflect.String:
		v.SetString(raw[0])
		return nil
	case reflect.Bool:
		b, err := strconv.ParseBool(raw[0])
		if err != nil {
			return err
		}
		v.SetBool(b)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(raw[0], 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetInt(i)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		u, err := strconv.ParseUint(raw[0], 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetUint(u)
		return nil
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(raw[0], v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetFloat(f)
		return nil
	default:
		return fmt.Errorf("不支持的字段类型: %s", v.Type())
	}
}
