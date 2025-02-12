package sql

import (
	"bytes"
	"os"
	"testing"
	"time"
)

func TestPackUnpack(t *testing.T) {
	type teststruct struct {
		A *int
		B uint
		C uint8
		D uint16
		E int32
		F uint32
		G int64
		H uint64
		I float32
		J float64
		K []byte
		L string
		M []string
		N *bool
		O *int8
	}
	_ = os.Remove("test.db")
	db := Sqlite{DBPath: "test.db"}
	err := db.Open(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	err = db.Create("test", &teststruct{})
	if err != nil {
		t.Fatal(err)
	}
	var o int8 = 2
	a := 2
	n := true
	inst := teststruct{&a, 2, 3, 4, 5, 6, 7, 8, 9.0, 10.0, []byte{1, 2, 3}, "123", []string{"123", "456"}, &n, nil}
	err = db.Insert("test", &inst)
	if err != nil {
		t.Fatal(err)
	}
	tmp := teststruct{O: &o}
	err = db.Find("test", &tmp, "")
	if err != nil {
		t.Fatal(err)
	}
	if *tmp.A != *inst.A {
		t.Fatal()
	}
	if tmp.B != inst.B {
		t.Fatal()
	}
	if tmp.C != inst.C {
		t.Fatal()
	}
	if tmp.D != inst.D {
		t.Fatal()
	}
	if tmp.E != inst.E {
		t.Fatal()
	}
	if tmp.F != inst.F {
		t.Fatal()
	}
	if tmp.F != inst.F {
		t.Fatal()
	}
	if tmp.G != inst.G {
		t.Fatal()
	}
	if tmp.H != inst.H {
		t.Fatal()
	}
	if tmp.I != inst.I {
		t.Fatal()
	}
	if tmp.J != inst.J {
		t.Fatal()
	}
	if !bytes.Equal(tmp.K, inst.K) {
		t.Fatal()
	}
	if tmp.L != inst.L {
		t.Fatal()
	}
	if tmp.M[0] != inst.M[0] {
		t.Fatal()
	}
	if *tmp.N != *inst.N {
		t.Fatal()
	}
	if tmp.O != nil {
		t.Fatal()
	}
	// 测试自增
	err = db.Insert("test", &teststruct{M: []string{""}})
	if err != nil {
		t.Fatal(err)
	}
	tmp = teststruct{O: &o}
	err = db.Find("test", &tmp, "WHERE A=3")
	if err != nil {
		t.Fatal(err)
	}
	if *tmp.A != 3 {
		t.Fatal()
	}
}

func TestFK(t *testing.T) {
	type teacher struct {
		ID   *int
		Name string
	}
	type class struct {
		ID           *int
		TeacherID    int
		StudentCount int
	}
	_ = os.Remove("test.db")
	db := Sqlite{DBPath: "test.db"}
	err := db.Open(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	err = db.Create("teacher", &teacher{})
	if err != nil {
		t.Fatal(err)
	}
	err = db.Insert("teacher", &teacher{Name: "Anna"}) // 1
	if err != nil {
		t.Fatal(err)
	}
	err = db.Insert("teacher", &teacher{Name: "Bob"}) // 2
	if err != nil {
		t.Fatal(err)
	}
	err = db.Insert("teacher", &teacher{Name: "Catalina"}) // 3
	if err != nil {
		t.Fatal(err)
	}
	err = db.Insert("teacher", &teacher{Name: "Donald"}) // 4
	if err != nil {
		t.Fatal(err)
	}
	err = db.Insert("teacher", &teacher{Name: "Emily"}) // 5
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.DB.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		t.Fatal(err)
	}
	err = db.Create("class", &class{}, "FOREIGN KEY(TeacherID) REFERENCES teacher(ID)")
	if err != nil {
		t.Fatal(err)
	}
	err = db.Insert("class", &class{TeacherID: 4, StudentCount: 66})
	if err != nil {
		t.Fatal(err)
	}
	err = db.Insert("class", &class{TeacherID: 3, StudentCount: 55})
	if err != nil {
		t.Fatal(err)
	}
	err = db.Insert("class", &class{TeacherID: 2, StudentCount: 44})
	if err != nil {
		t.Fatal(err)
	}
	err = db.Insert("class", &class{TeacherID: 1, StudentCount: 33})
	if err != nil {
		t.Fatal(err)
	}
	err = db.Insert("class", &class{TeacherID: 5, StudentCount: 22})
	if err != nil {
		t.Fatal(err)
	}
	err = db.Insert("class", &class{TeacherID: 6, StudentCount: 11})
	if err == nil {
		t.Fatal("unexpected success")
	}
}
