/*
Copyright 2013 Tamás Gulácsi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kylelemons/godebug/pretty"
)

// TestGen tests the generation - for this, it needs a dsn with privileges
// if you get "ORA-01031: insufficient privileges", then you need
// GRANT CREATE PROCEDURE TO username;
func TestGenSimple(t *testing.T) {
	build(t)
	outFn := generateAndBuild(t, "SIMPLE_")

	for i, todo := range [][3]string{
		{"simple_char_in", `{"txt": "abraka dabra"}`, `{}`},
		{"simple_char_out", `{}`, `{"ret":"A"}`},
		{"simple_num_in", `{"num": 33}`, `{}`},
		{"simple_num_out", `{}`, `{"ret":0.6666666666666666}`},
		{"simple_date_in", `{"dat": "2013-12-25T21:15:00+01:00"}`, `{}`},
		{"simple_date_out", `{}`, `{"ret":"{{NOW}}"}`}, // 5.
		{"simple_char_in_char_ret", `{"txt": "abraka dabra"}`, `{"ret":"Typ=1 Len=12: 97,98,114,97,107,97,32,100,97,98,114,97"}`},
		{"simple_all_inout",
			`{"txt1": "abraka", "txt3": "A", "int1": -1, "int3": -2, "num1": 0.1, "num3": 0.3, "dt1": null, "dt3": "2014-01-03T00:00:00+01:00"}`,
			`{"txt2":"abraka#","int2":0,"num2":0.43333333333333335,"dt2":"0000-01-31T00:00:00+01:00","txt3":"A#","int3":-1,"num3":1.3,"dt3":"2014-02-03T00:00:00+01:00"}`},
		{"simple_nums_count", `{"nums":[1,2,3,4.4]}`, `{"ret":4}`},
		{"simple_sum_nums", `{"nums":[1.1,2,3.3]}`, `{"outnums":[2.1,3,4.3],"ret":6.4}`},
	} {
		got := runTest(t, outFn, "-connect="+*flagConnect, "TST_oracall."+todo[0], todo[1])
		if strings.Index(todo[2], "{{NOW}}") >= 0 {
			todo[2] = strings.Replace(todo[2], "{{NOW}}", time.Now().Format(time.RFC3339), -1)
		}
		if strings.TrimSpace(got) != todo[2] {
			t.Errorf("%d. awaited\n\t%s\ngot\n\t%s", i, todo[2], got)
		}
	}
}

func TestGenRec(t *testing.T) {
	build(t)
	outFn := generateAndBuild(t, "REC_")

	for i, todo := range [][3]string{
		{"rec_in", `{"rec":{"num":33,"text":"xxx","dt":"2006-08-26T00:00:00+01:00"}}`,
			`{"ret":"33;\"2006-08-26 00:00:00\";\"xxx\""}`},
		{"rec_tab_in", `{"tab":[{"num":1,"text":"A","dt":"2006-08-26T00:00:00+01:00"},{"num":2,"text":"B"},{"num":3,"text":"C"}]}`,
			`{"ret":"\n1;\"2006-08-26 00:00:00\";\"A\"\n2;\"0001-01-01 00:00:00\";\"B\"\n3;\"0001-01-01 00:00:00\";\"C\""}`},
		{"rec_sendpreoffer_31101", `{"p_vonalkod":1}`,
			`{"p_vonalkod":1,"p_kotveny":{"szerkot":"0001-01-01T00:00:00Z","halaszt_kockezd":"0001-01-01T00:00:00Z","halaszt_dijfiz":"0001-01-01T00:00:00Z","szamlaszam":"","szamla_limit":0,"e_komm_email":"","evfordulo":"0001-01-01T00:00:00Z","evfordulo_tipus":"","dijbekerot_ker":"","ajanlati_evesdij":0,"kockezd":"0001-01-01T00:00:00Z","btkezd":"0001-01-01T00:00:00Z","dijkod":"","dijfizmod":"","dijfizgyak":"","szerlejar":"0001-01-01T00:00:00Z"},"p_kotveny_gfb":{"bm_tipus":0,"kotes_oka":0},"p_gepjarmu":{"jelleg":"","rendszam":"","gyartev":"","tulajdon_visz":"","teljesitmeny":0,"ossztomeg":0,"ferohely":0,"uzjelleg":"","alvazszam":"","gyartmany":"","tulajdon_ido":"0001-01-01T00:00:00Z"},"p_szerz_azon":0,"p_ajanlat_url":"","p_evesdij":0,"p_hiba_kod":0}`},
	} {
		got := runTest(t, outFn, "-connect="+*flagConnect, "TST_oracall."+todo[0], todo[1])
		if strings.Index(todo[2], "{{NOW}}") >= 0 {
			todo[2] = strings.Replace(todo[2], "{{NOW}}", time.Now().Format(time.RFC3339), -1)
		}
		if diff := jsonEqual(strings.TrimSpace(got), todo[2]); diff != "" {
			t.Errorf("%d. awaited\n\t%s\ngot\n\t%s\ndiff\n\t%s", i, todo[2], got, diff)
		}
	}
}

func createStoredProc(t *testing.T) {
	compile(t, `CREATE OR REPLACE PACKAGE TST_oracall AS
TYPE num_tab_typ IS TABLE OF NUMBER INDEX BY BINARY_INTEGER;

TYPE mix_rec_typ IS RECORD (num NUMBER, dt DATE, text VARCHAR2(1000));
TYPE mix_tab_typ IS TABLE OF mix_rec_typ INDEX BY BINARY_INTEGER;

  TYPE kotveny_rec_typ IS RECORD (
    dijkod VARCHAR2(2),
    dijfizmod VARCHAR2(1),
    dijfizgyak VARCHAR2(1),
    szerkot DATE,
    szerlejar DATE,
    kockezd DATE,
    btkezd DATE,
    halaszt_kockezd DATE,
    halaszt_dijfiz DATE,
    szamlaszam VARCHAR2(24),
    szamla_limit  NUMBER(12,2),
    evfordulo DATE,
    evfordulo_tipus VARCHAR2(1),
    e_komm_email VARCHAR2(100),
    dijbekerot_ker VARCHAR2(1),
    ajanlati_evesdij NUMBER(15,2)
  );
  TYPE mod_ugyfel_rec_typ IS RECORD (
    nem VARCHAR2(1),
    ugyfelnev VARCHAR2(40),
    szulnev VARCHAR2(40),
    anyanev VARCHAR2(40),
    szulhely VARCHAR2(27),
    szuldat DATE,
    adoszam VARCHAR2(11),
    adoaz VARCHAR2(13),
    aht_azon NUMBER(6),
    szemelyaz VARCHAR2(8),
    jogositvany_kelte DATE,
    tel_otthon VARCHAR2(12),
    tel_mhely VARCHAR2(12),
    tel_mobil VARCHAR2(12),
    email VARCHAR2(100),
    tbazon VARCHAR2(10),
    vallalkozo VARCHAR2(1),
    okmany_tip VARCHAR2(6),
    okmanyszam VARCHAR2(25),
    adatkez VARCHAR2(1),
    cegkepviselo_neve VARCHAR2(40),
    cegforma VARCHAR2(1),
    cegjegyzekszam VARCHAR2(25),
    allampolg VARCHAR2(1),
    tart_eng VARCHAR2(1),
    publikus VARCHAR2(1)
  );
  TYPE mod_ugyfel_tab_typ IS TABLE OF mod_ugyfel_rec_typ INDEX BY PLS_INTEGER;
  TYPE mod_cim_rec_typ IS RECORD (
    ktid PLS_INTEGER,
    hazszam1 VARCHAR2(5),
    hazszam2 VARCHAR2(5),
    epulet VARCHAR2(5),
    lepcsohaz VARCHAR2(5),
    emelet VARCHAR2(5),
    ajto VARCHAR2(5),
    hrsz VARCHAR2(5),
    kulf_orsz_kod VARCHAR2(3),
    kulf_irszam VARCHAR2(5),
    kulf_helynev VARCHAR2(5),
    kulf_utca VARCHAR2(5),
    kulf_hszajto VARCHAR2(5),
    kulf_egyeb VARCHAR2(5),
    kulf_pf VARCHAR2(5)
  );
  TYPE mod_cim_tab_typ IS TABLE OF mod_cim_rec_typ INDEX BY PLS_INTEGER;
  TYPE engedmeny_rec_typ IS RECORD (
    engedm_kezdet DATE,
    engedm_veg DATE,
    vagyontargy VARCHAR2(25),
    engedm_osszeg nUMBER(12,2),
    also_limit nUMBER(12,2),
    felso_limit nUMBER(12,2),
    penznem VARCHAR2(3),
    szamlaszam VARCHAR2(24),
    hitel_szam VARCHAR2(10),
    fed_bejegy VARCHAR2(10),
    bizt_nyil VARCHAR2(10)
  );
  TYPE engedmeny_tab_typ IS TABLE OF engedmeny_rec_typ INDEX BY PLS_INTEGER;
  TYPE kotveny_gfb_rec_typ IS RECORD (
    bm_tipus PLS_INTEGER,
    kotes_oka PLS_INTEGER
  );
  TYPE gepjarmu_rec_typ IS RECORD (
    jelleg VARCHAR2(2),
    rendszam VARCHAR2(11),
    teljesitmeny PLS_INTEGER,
    ossztomeg PLS_INTEGER,
    ferohely PLS_INTEGER,
    uzjelleg VARCHAR2(1),
    alvazszam VARCHAR2(40),
    gyartev VARCHAR2(4),
    gyartmany VARCHAR2(20),
    tulajdon_ido DATE,
    tulajdon_visz VARCHAR2(1)
  );
  TYPE bonusz_elozmeny_rec_typ IS RECORD (
    zaro_bonusz VARCHAR2(3),
    kov_bonusz VARCHAR2(3),
    rendszam VARCHAR2(11),
    bizt_kod VARCHAR2(2),
    kotvenyszam VARCHAR2(24),
    torles_oka VARCHAR2(2),
    szerkot DATE,
    szervege DATE
  );
  TYPE kedvezmeny_tab_typ IS TABLE OF VARCHAR2(6) INDEX BY PLS_INTEGER;
  TYPE kedv_modkod_tab_typ IS TABLE OF VARCHAR2(1) INDEX BY VARCHAR2(6);
  TYPE nevszam_rec_typ IS RECORD (nev VARCHAR2(80), ertek NUMBER(12,2));
  TYPE nevszam_tab_typ IS TABLE OF nevszam_rec_typ INDEX BY PLS_INTEGER;
  TYPE hiba_rec_typ IS RECORD (hibaszam PLS_INTEGER, szoveg VARCHAR2(100));
  TYPE hiba_tab_typ IS TABLE OF hiba_rec_typ INDEX BY PLS_INTEGER;

PROCEDURE simple_char_in(txt IN VARCHAR2);
FUNCTION simple_char_out RETURN VARCHAR2;
PROCEDURE simple_num_in(num IN NUMBER);
FUNCTION simple_num_out RETURN NUMBER;
PROCEDURE simple_date_in(dat IN DATE);
FUNCTION simple_date_out RETURN DATE;
FUNCTION simple_char_in_char_ret(txt IN VARCHAR2) RETURN VARCHAR2;
PROCEDURE simple_all_inout(
    txt1 IN VARCHAR2, int1 IN PLS_INTEGER, num1 IN NUMBER, dt1 IN DATE,
    txt2 OUT VARCHAR2, int2 OUT PLS_INTEGER, num2 OUT NUMBER, dt2 OUT DATE,
    txt3 IN OUT VARCHAR2, int3 IN OUT PLS_INTEGER, num3 IN OUT NUMBER, dt3 IN OUT DATE);

FUNCTION simple_nums_count(nums IN num_tab_typ) RETURN PLS_INTEGER;
FUNCTION simple_sum_nums(nums IN num_tab_typ, outnums OUT num_tab_typ) RETURN NUMBER;

FUNCTION rec_in(rec IN mix_rec_typ) RETURN VARCHAR2;
FUNCTION rec_tab_in(tab IN mix_tab_typ) RETURN VARCHAR2;

  PROCEDURE rec_sendPreOffer_31101(p_sessionid IN VARCHAR2,
                               p_lang IN VARCHAR2,
                               p_vegleges IN VARCHAR2 DEFAULT 'N',
                               p_elso_csekk_atadva IN VARCHAR2 DEFAULT 'N',
                               p_vonalkod IN OUT PLS_INTEGER,
                               p_kotveny IN OUT kotveny_rec_typ,
                               p_szerzodo IN mod_ugyfel_rec_typ,
                               p_szerzodo_cim IN mod_cim_rec_typ,
                               p_szerzodo_levelcim IN mod_cim_rec_typ,
                               p_engedmenyezett IN mod_ugyfel_rec_typ,
                               p_engedmenyezett_cim IN mod_cim_rec_typ,
                               p_engedmeny IN engedmeny_rec_typ,
                               p_kotveny_gfb IN OUT NOCOPY kotveny_gfb_rec_typ,
                               p_gepjarmu IN OUT gepjarmu_rec_typ,
                               p_bonusz_elozmeny IN bonusz_elozmeny_rec_typ,
--                               p_kedvezmenyek IN kedvezmeny_tab_typ,
                               p_dump_args# IN VARCHAR2,
                               p_szerz_azon OUT PLS_INTEGER,
                               p_ajanlat_url OUT VARCHAR2,
                               p_szamolt_dijtetelek OUT nevszam_tab_typ,

                               p_evesdij OUT NUMBER,
                               p_hibalista OUT hiba_tab_typ,
                               p_hiba_kod OUT PLS_INTEGER,
                               p_hiba_szov OUT VARCHAR2);

END TST_oracall;`)

	compile(t, `CREATE OR REPLACE PACKAGE BODY TST_oracall AS
PROCEDURE simple_char_in(txt IN VARCHAR2) IS
  v_txt VARCHAR2(1000) := SUBSTR(txt, 1, 100);
BEGIN NULL; END simple_char_in;
FUNCTION simple_char_out RETURN VArCHAR2 IS BEGIN RETURN('A'); END simple_char_out;

PROCEDURE simple_num_in(num IN NUMBER) IS
  v_num NUMBER := num;
BEGIN NULL; END simple_num_in;

PROCEDURE simple_date_in(dat IN DATE) IS
  v_dat DATE := dat;
BEGIN NULL; END simple_date_in;

FUNCTION simple_char_in_char_ret(txt IN VARCHAR2) RETURN VARCHAR2 IS
  v_txt CONSTANT VARCHAR2(4000) := SUBSTR(txt, 1, 4000);
  v_ret VARCHAR2(4000);
BEGIN
  SELECT DUMP(txt) INTO v_ret FROM DUAL;
  RETURN v_ret;
END simple_char_in_char_ret;

FUNCTION simple_date_out RETURN DATE IS BEGIN RETURN SYSDATE; END simple_date_out;
FUNCTION simple_num_out RETURN NUMBER IS BEGIN RETURN 2/3; END simple_num_out;

PROCEDURE simple_all_inout(
    txt1 IN VARCHAR2, int1 IN PLS_INTEGER, num1 IN NUMBER, dt1 IN DATE,
    txt2 OUT VARCHAR2, int2 OUT PLS_INTEGER, num2 OUT NUMBER, dt2 OUT DATE,
    txt3 IN OUT VARCHAR2,
    int3 IN OUT PLS_INTEGER, num3 IN OUT NUMBER, dt3 IN OUT DATE) IS
BEGIN
  txt2 := txt1||'#'; int2 := NVL(int1, 0) + 1;
  num2 := NVL(num1, 0) + 1/3; dt2 := ADD_MONTHS(NVL(dt1, SYSDATE), 1);
  txt3 := txt3||'#'; int3 := NVL(int3, 0) + 1;
  num3 := NVL(num3, 0) + 1; dt3 := ADD_MONTHS(NVL(dt3, SYSDATE), 1);
END simple_all_inout;

FUNCTION simple_nums_count(nums IN num_tab_typ) RETURN PLS_INTEGER IS
BEGIN
  RETURN nums.COUNT;
END simple_nums_count;

FUNCTION simple_sum_nums(nums IN num_tab_typ, outnums OUT num_tab_typ) RETURN NUMBER IS
  v_idx PLS_INTEGER;
  s NUMBER := 0;
BEGIN
  outnums.DELETE;
  v_idx := nums.FIRST;
  WHILE v_idx IS NOT NULL LOOP
    s := NVL(s, 0) + NVL(nums(v_idx), 0);
    outnums(v_idx) := NVL(nums(v_idx), 0) + 1;
    v_idx := nums.NEXT(v_idx);
  END LOOP;
  RETURN(s);
END simple_sum_nums;

FUNCTION rec_in(rec IN mix_rec_typ) RETURN VARCHAR2 IS
BEGIN
  RETURN rec.num||';"'||TO_CHAR(rec.dt, 'YYYY-MM-DD HH24:MI:SS')||'";"'||rec.text||'"';
END rec_in;

FUNCTION rec_tab_in(tab IN mix_tab_typ) RETURN VARCHAR2 IS
  i PLS_INTEGER;
  text VARCHAR2(32767);
BEGIN
  i := tab.FIRST;
  WHILE i IS NOT NULL LOOP
    text := text||CHR(10)||SUBSTR(
              tab(i).num||';"'||TO_CHAR(tab(i).dt, 'YYYY-MM-DD HH24:MI:SS')
              ||'";"'||tab(i).text||'"',
              1, GREATEST(0, 32767-NVL(LENGTH(text), 0)-1));
    EXIT WHEN LENGTH(text) >= 32767;
    i := tab.NEXT(i);
  END LOOP;
  RETURN(text);
END rec_tab_in;

FUNCTION nums_count(nums IN num_tab_typ) RETURN PLS_INTEGER IS
BEGIN
  RETURN nums.COUNT;
END nums_count;

FUNCTION sum_nums(nums IN num_tab_typ, outnums OUT num_tab_typ) RETURN NUMBER IS
  v_idx PLS_INTEGER;
  s NUMBER := 0;
BEGIN
  outnums.DELETE;
  v_idx := nums.FIRST;
  WHILE v_idx IS NOT NULL LOOP
    s := NVL(s, 0) + NVL(nums(v_idx), 0);
    outnums(v_idx) := NVL(nums(v_idx), 0) * 2;
    v_idx := nums.NEXT(v_idx);
  END LOOP;
  RETURN(s);
END sum_nums;

  PROCEDURE rec_sendPreOffer_31101(p_sessionid IN VARCHAR2,
                               p_lang IN VARCHAR2,
                               p_vegleges IN VARCHAR2 DEFAULT 'N',
                               p_elso_csekk_atadva IN VARCHAR2 DEFAULT 'N',
                               p_vonalkod IN OUT PLS_INTEGER,
                               p_kotveny IN OUT kotveny_rec_typ,
                               p_szerzodo IN mod_ugyfel_rec_typ,
                               p_szerzodo_cim IN mod_cim_rec_typ,
                               p_szerzodo_levelcim IN mod_cim_rec_typ,
                               p_engedmenyezett IN mod_ugyfel_rec_typ,
                               p_engedmenyezett_cim IN mod_cim_rec_typ,
                               p_engedmeny IN engedmeny_rec_typ,
                               p_kotveny_gfb IN OUT NOCOPY kotveny_gfb_rec_typ,
                               p_gepjarmu IN OUT gepjarmu_rec_typ,
                               p_bonusz_elozmeny IN bonusz_elozmeny_rec_typ,
--                               p_kedvezmenyek IN kedvezmeny_tab_typ,
                               p_dump_args# IN VARCHAR2,
                               p_szerz_azon OUT PLS_INTEGER,
                               p_ajanlat_url OUT VARCHAR2,
                               p_szamolt_dijtetelek OUT nevszam_tab_typ,
                               p_evesdij OUT NUMBER,
                               p_hibalista OUT hiba_tab_typ,
                               p_hiba_kod OUT PLS_INTEGER,
                               p_hiba_szov OUT VARCHAR2) IS
  BEGIN
    p_hiba_kod := 0; p_hiba_szov := NULL;
  END rec_sendPreOffer_31101;

END TST_oracall;`)
}

func compile(t *testing.T, qry string) {
	typ := "PACKAGE"
	if strings.HasPrefix(qry, "CREATE OR REPLACE PACKAGE BODY") {
		typ = typ + " BODY"
	}
	db := getConnection(t)
	if _, err := db.Exec(qry); err != nil {
		log.Println(qry)
		t.Fatalf("error creating %s: %v", typ, err)
	}

	rows, err := db.Query(`SELECT line||'/'||position||': '||text
          FROM user_errors WHERE type = :1 AND name = :2`,
		typ, "TST_ORACALL")
	if err != nil {
		t.Fatalf("error querying errors: %v", err)
	}
	defer rows.Close()
	var str string
	var errTexts []string
	for rows.Next() {
		if err = rows.Scan(&str); err != nil {
			t.Fatalf("error fetching: %v", err)
		}
		errTexts = append(errTexts, str)
	}
	if len(errTexts) > 0 {
		fmt.Println(qry)
		fmt.Println("/")
		t.Fatalf("error with package: %s", strings.Join(errTexts, "\n"))
	}
}

func runCommand(t *testing.T, prog string, args ...string) {
	out, err := exec.Command(prog, args...).CombinedOutput()
	if err != nil {
		t.Errorf("error '%q %s': %v\n%s", prog, args, err, out)
		t.FailNow()
	} else {
		t.Logf("%q %s:\n%s", prog, args, out)
	}
}

func build(t *testing.T) {
	buildOnce.Do(func() {
		createStoredProc(t)
		runCommand(t, "go", "build")
	})
}

func generateAndBuild(t *testing.T, prefix string) (outFn string) {
	runCommand(t, "sh", "-c",
		"./oracall -F -connect='"+*flagConnect+
			"' TST_ORACALL."+strings.ToUpper(prefix)+"%"+
			" >./testdata/integration_test/generated_functions.go")

	if outFh, err := ioutil.TempFile("", "oracall-integration_test"); err != nil {
		t.Errorf("cannot create temp file: %v", err)
		t.FailNow()
	} else {
		outFn = outFh.Name()
		outFh.Close()
	}
	os.Remove(outFn)
	runCommand(t, "go", "build", "-o="+outFn, "./testdata/integration_test")
	return
}

var errBuf = bytes.NewBuffer(make([]byte, 0, 512))

func runTest(t *testing.T, prog string, args ...string) string {
	c := exec.Command(prog, args...)
	errBuf.Reset()
	c.Stderr = errBuf
	out, err := c.Output()
	if err != nil {
		t.Errorf("error '%q %s': %v\n%s", prog, args, err, errBuf)
		t.FailNow()
	} else {
		t.Logf("%q %s:\n%s\n%s", prog, args, out, errBuf)
	}
	return string(out)
}

//var dsn = flag.String("connect", "", "Oracle DSN (user/passw@sid)")
var dbg = flag.Bool("debug", false, "print debug messages?")
var buildOnce sync.Once

func init() {
	flag.Parse()
}

var db *sql.DB

func getConnection(t *testing.T) *sql.DB {
	if db != nil && db.Ping() == nil {
		return db
	}

	if !(flagConnect != nil && *flagConnect != "") {
		t.Logf("cannot test connection without dsn!")
		t.FailNow()
	}
	var err error
	db, err = sql.Open("gocilib", *flagConnect)
	if err != nil {
		log.Panicf("error creating connection to %s: %s", *flagConnect, err)
	}
	return db
}

func jsonEqual(a, b string) string {
	var aJ, bJ map[string]interface{}
	if err := json.Unmarshal([]byte(a), &aJ); err != nil {
		return "ERROR a: " + err.Error()
	}
	omitEmpty(aJ)
	if err := json.Unmarshal([]byte(b), &bJ); err != nil {
		return "ERROR b: " + err.Error()
	}
	omitEmpty(bJ)
	return pretty.Compare(aJ, bJ)
}

func omitEmpty(m map[string]interface{}) {
	for k, v := range m {
		empty := false
		switch x := v.(type) {
		case string:
			empty = x == ""
		case int:
			empty = x == 0
		case float64:
			empty = x == 0
		case time.Time:
			empty = x.IsZero()
		case map[string]interface{}:
			omitEmpty(x)
			empty = len(x) == 0
		}
		if empty {
			delete(m, k)
		}
	}
}
