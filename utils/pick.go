package main

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
)

const OutputFieldName = false // User{ID:1,Name:"aa"} にするか User{1,"aa"} にするか

func main() {
	pick()
}
func pick() {

	// 書きやすい
	// "github.com/jmoiron/sqlx"
	dsn := fmt.Sprintf("%s%s@tcp(%s:%s)/isubata?parseTime=true&loc=Local&charset=utf8mb4",
		"isucon", ":isucon", "127.0.0.1", "3306")
	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	type Image struct {
		ID   int64  `db:"id"`
		name string `db:"name"`
		data []byte `db:"data`
	}
	// Image : ID(>1001 は消える.Nameで取得.)/Name/DATA
	images := []Image{}
	err = db.Select(&images, "SELECT name, data FROM image WHERE ID <= 1001")
	if err != nil {
		panic(err)
	}
	for _, image := range images {
		file, err := os.Create(image.name)
		if err != nil {
			log.Panic(err)
		}
		file.Write(image.data)
		file.Close()
	}

}

func Normalize(x reflect.Type) string {
	return strings.Replace(fmt.Sprint(x), "main.", "", -1)
}

// 基本は Printf %#v でよいのだが、{,*}time.Time など,時々変なものがあるので
func PrintRecursive(v reflect.Value, indent int, uniqName string) {
	t := v.Type()
	tName := Normalize(t)
	switch t.Kind() {
	case reflect.Slice:
		if v.Len() == 0 {
			fmt.Print(tName, "{}")
		} else {
			fmt.Println(tName, "{")
			for i := 0; i < v.Len(); i++ {
				for j := 0; j <= indent; j++ {
					fmt.Print("\t")
				}
				PrintRecursive(v.Index(i), indent+1, uniqName)
				fmt.Println(",")
			}
			fmt.Print("}")
		}
	case reflect.Struct:
		switch tName {
		case "time.Time":
			fmt.Print(`timeFor`, uniqName, `("`, v, `")`)
		default: // 普通の構造体
			fmt.Print(tName, "{")
			for i := 0; i < t.NumField(); i++ {
				if OutputFieldName {
					fmt.Print(t.Field(i).Name, ":")
				}
				PrintRecursive(v.Field(i), indent+1, uniqName)
				fmt.Print(",")
			}
			fmt.Print("}")
		}
	case reflect.Ptr:
		switch tName {
		case "*time.Time":
			fmt.Print(`timeFor`, uniqName, `Ptr("`, v, `")`)
		default:
			log.Panic("\nNOT SUPPORTED PTR:", tName)
		}
	case reflect.Map:
		if v.Len() == 0 {
			fmt.Print(tName, "{}")
		} else {
			fmt.Println(tName, "{")
			keys := v.MapKeys()
			sort.Slice(keys, func(i, j int) bool {
				return fmt.Sprint(keys[i]) < fmt.Sprint(keys[j])
			})
			for _, mk := range keys {
				mv := v.MapIndex(mk)
				for j := 0; j <= indent; j++ {
					fmt.Print("\t")
				}
				fmt.Printf("%#v:", mk)
				PrintRecursive(mv, indent+1, uniqName)
				fmt.Println()
			}
			fmt.Print("}")
		}
	default:
		fmt.Printf("%#v", v)
	}
}

func Output(targetName string, targets interface{}) {
	fmt.Print(`
package main
import "time"
func timeFor` + targetName + `(s string) time.Time {
	result, _ := time.Parse("2006-01-02 15:04:05 -0700 MST", s)
	return result
}
func timeFor` + targetName + `Ptr(s string) *time.Time {
	result, _ := time.Parse("2006-01-02 15:04:05 -0700 MST", s)
	return &result
}
`)
	fmt.Print("var ", targetName, " = ")
	PrintRecursive(reflect.ValueOf(targets), 0, targetName)
	fmt.Println()
}
