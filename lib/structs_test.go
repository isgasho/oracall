/*
Copyright 2015 Tamás Gulácsi

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

package oracall

import (
	"strings"
	"testing"
)

var testCases = []testCase{
	{Csv: `OBJECT_ID;SUBPROGRAM_ID;PACKAGE_NAME;OBJECT_NAME;DATA_LEVEL;POSITION;ARGUMENT_NAME;IN_OUT;DATA_TYPE;DATA_PRECISION;DATA_SCALE;CHARACTER_SET_NAME;PLS_TYPE;CHAR_LENGTH;TYPE_LINK;TYPE_OWNER;TYPE_NAME;TYPE_SUBNAME
19734;35;DB_WEB;SENDPREOFFER_31101;0;1;P_SESSIONID;IN/OUT;VARCHAR2;;;CHAR_CS;VARCHAR2;;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;0;5;P_VONALKOD;IN/OUT;BINARY_INTEGER;;;;PLS_INTEGER;0;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;0;1;DIJKOD;IN/OUT;CHAR;;;CHAR_CS;CHAR;2;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;0;4;SZERKOT;IN/OUT;DATE;;;;DATE;0;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;0;16;AJANLATI_EVESDIJ;IN/OUT;NUMBER;12;2;;NUMBER;0;;;;
`,
		PlSql: `DECLARE
  i1 PLS_INTEGER;
  i2 PLS_INTEGER;

BEGIN

  DB_web.sendpreoffer_31101(p_sessionid=>:1,
                p_vonalkod=>:2,
                dijkod=>:3,
                szerkot=>:4,
                ajanlati_evesdij=>:5);


END;
`},

	{Csv: `OBJECT_ID;SUBPROGRAM_ID;PACKAGE_NAME;OBJECT_NAME;DATA_LEVEL;POSITION;ARGUMENT_NAME;IN_OUT;DATA_TYPE;DATA_PRECISION;DATA_SCALE;CHARACTER_SET_NAME;PLS_TYPE;CHAR_LENGTH;TYPE_LINK;TYPE_OWNER;TYPE_NAME;TYPE_SUBNAME
19734;35;DB_WEB;SENDPREOFFER_31101;0;6;P_KOTVENY;IN/OUT;PL/SQL RECORD;;;;;0;;BRUNO;DB_WEB_ELEKTR;KOTVENY_REC_TYP
19734;35;DB_WEB;SENDPREOFFER_31101;1;1;DIJKOD;IN/OUT;CHAR;;;CHAR_CS;CHAR;2;;;;
`,
	},

	{Csv: `OBJECT_ID;SUBPROGRAM_ID;PACKAGE_NAME;OBJECT_NAME;DATA_LEVEL;POSITION;ARGUMENT_NAME;IN_OUT;DATA_TYPE;DATA_PRECISION;DATA_SCALE;CHARACTER_SET_NAME;PLS_TYPE;CHAR_LENGTH;TYPE_LINK;TYPE_OWNER;TYPE_NAME;TYPE_SUBNAME
19734;35;DB_WEB;SENDPREOFFER_31101;0;1;P_SESSIONID;IN;VARCHAR2;;;CHAR_CS;VARCHAR2;;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;0;2;P_LANG;IN;VARCHAR2;;;CHAR_CS;VARCHAR2;;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;0;3;P_VEGLEGES;IN;VARCHAR2;;;CHAR_CS;VARCHAR2;;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;0;4;P_ELSO_CSEKK_ATADVA;IN;VARCHAR2;;;CHAR_CS;VARCHAR2;;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;0;5;P_VONALKOD;IN/OUT;BINARY_INTEGER;;;;PLS_INTEGER;0;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;0;6;P_KOTVENY;IN/OUT;PL/SQL RECORD;;;;;0;;BRUNO;DB_WEB_ELEKTR;KOTVENY_REC_TYP
19734;35;DB_WEB;SENDPREOFFER_31101;1;1;DIJKOD;IN/OUT;CHAR;;;CHAR_CS;CHAR;2;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;1;2;DIJFIZMOD;IN/OUT;CHAR;;;CHAR_CS;CHAR;1;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;1;3;DIJFIZGYAK;IN/OUT;CHAR;;;CHAR_CS;CHAR;1;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;1;4;SZERKOT;IN/OUT;DATE;;;;DATE;0;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;1;5;SZERLEJAR;IN/OUT;DATE;;;;DATE;0;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;1;6;KOCKEZD;IN/OUT;DATE;;;;DATE;0;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;1;7;BTKEZD;IN/OUT;DATE;;;;DATE;0;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;1;8;HALASZT_KOCKEZD;IN/OUT;DATE;;;;DATE;0;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;1;9;HALASZT_DIJFIZ;IN/OUT;DATE;;;;DATE;0;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;1;10;SZAMLASZAM;IN/OUT;VARCHAR2;;;CHAR_CS;VARCHAR2;24;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;1;11;SZAMLA_LIMIT;IN/OUT;NUMBER;12;2;;NUMBER;0;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;1;12;EVFORDULO;IN/OUT;DATE;;;;DATE;0;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;1;13;EVFORDULO_TIPUS;IN/OUT;VARCHAR2;;;CHAR_CS;VARCHAR2;1;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;1;14;E_KOMM_EMAIL;IN/OUT;VARCHAR2;;;CHAR_CS;VARCHAR2;80;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;1;15;DIJBEKEROT_KER;IN/OUT;VARCHAR2;;;CHAR_CS;VARCHAR2;1;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;1;16;AJANLATI_EVESDIJ;IN/OUT;NUMBER;12;2;;NUMBER;0;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;0;16;P_KEDVEZMENYEK;IN;PL/SQL TABLE;;;;;0;;BRUNO;DB_WEB_ELEKTR;KEDVEZMENY_TAB_TYP
19734;35;DB_WEB;SENDPREOFFER_31101;1;1;;IN;VARCHAR2;;;CHAR_CS;VARCHAR2;6;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;0;17;P_DUMP_ARGS#;IN;VARCHAR2;;;CHAR_CS;VARCHAR2;;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;0;18;P_SZERZ_AZON;OUT;BINARY_INTEGER;;;;PLS_INTEGER;0;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;0;19;P_AJANLAT_URL;OUT;VARCHAR2;;;CHAR_CS;VARCHAR2;;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;0;20;P_SZAMOLT_DIJTETELEK;OUT;PL/SQL TABLE;;;;;0;;BRUNO;DB_WEB_PORTAL;NEVSZAM_TAB_TYP
19734;35;DB_WEB;SENDPREOFFER_31101;1;1;;OUT;PL/SQL RECORD;;;;;0;;BRUNO;DB_WEB_PORTAL;NEVSZAM_REC_TYP
19734;35;DB_WEB;SENDPREOFFER_31101;2;1;NEV;OUT;VARCHAR2;;;CHAR_CS;VARCHAR2;80;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;2;2;ERTEK;OUT;NUMBER;12;2;;NUMBER;0;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;0;21;P_EVESDIJ;OUT;NUMBER;;;;NUMBER;0;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;0;22;P_HIBALISTA;OUT;PL/SQL TABLE;;;;;0;;BRUNO;DB_WEB_ELEKTR;HIBA_TAB_TYP
19734;35;DB_WEB;SENDPREOFFER_31101;1;1;;OUT;PL/SQL RECORD;;;;;0;;BRUNO;DB_WEB_ELEKTR;HIBA_REC_TYP
19734;35;DB_WEB;SENDPREOFFER_31101;2;1;HIBASZAM;OUT;NUMBER;9;;;NUMBER;0;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;2;2;SZOVEG;OUT;VARCHAR2;;;CHAR_CS;VARCHAR2;1000;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;0;23;P_HIBA_KOD;OUT;BINARY_INTEGER;;;;PLS_INTEGER;0;;;;
19734;35;DB_WEB;SENDPREOFFER_31101;0;24;P_HIBA_SZOV;OUT;VARCHAR2;;;CHAR_CS;VARCHAR2;;;;;
`,
		PlSql: `DECLARE
TYPE NUMBER_12__2_tab_typ IS TABLE OF NUMBER(12, 2) INDEX BY BINARY_INTEGER;
  TYPE VARCHAR2_80_tab_typ IS TABLE OF VARCHAR2(80) INDEX BY BINARY_INTEGER;
  TYPE NUMBER_9_tab_typ IS TABLE OF NUMBER(9) INDEX BY BINARY_INTEGER;
  TYPE VARCHAR2_1000_tab_typ IS TABLE OF VARCHAR2(1000) INDEX BY BINARY_INTEGER;
  v001 BRUNO.DB_WEB_ELEKTR.KOTVENY_REC_TYP;
    x021# DB_WEB_PORTAL.NEVSZAM_TAB_TYP;
  x021#_idx PLS_INTEGER := NULL;
  x021#nev VARCHAR2_80_tab_typ;
  x021#ertek NUMBER_12__2_tab_typ;
    x026# DB_WEB_ELEKTR.HIBA_TAB_TYP;
  x026#_idx PLS_INTEGER := NULL;
  x026#hibaszam NUMBER_9_tab_typ;
  x026#szoveg VARCHAR2_1000_tab_typ;
BEGIN

  v001.dijkod := :x002#dijkod;
  v001.dijfizmod := :x002#dijfizmod;
  v001.dijfizgyak := :x002#dijfizgyak;
  v001.szerkot := :x002#szerkot;
  v001.szerlejar := :x002#szerlejar;
  v001.kockezd := :x002#kockezd;
  v001.btkezd := :x002#btkezd;
  v001.halaszt_kockezd := :x002#halaszt_kockezd;
  v001.halaszt_dijfiz := :x002#halaszt_dijfiz;
  v001.szamlaszam := :x002#szamlaszam;
  v001.szamla_limit := :x002#szamla_limit;
  v001.evfordulo := :x002#evfordulo;
  v001.evfordulo_tipus := :x002#evfordulo_tipus;
  v001.e_komm_email := :x002#e_komm_email;
  v001.dijbekerot_ker := :x002#dijbekerot_ker;
  v001.ajanlati_evesdij := :x002#ajanlati_evesdij;

  x021#.DELETE;
  x021#nev.DELETE; x021#ertek.DELETE;

  x026#.DELETE;
  x026#szoveg.DELETE; x026#hibaszam.DELETE;

  DB_web.sendpreoffer_31101(p_sessionid=>:p_sessionid,
               p_lang=>:p_lang,
               p_vonalkod=>:p_vonalkod,
               p_kotveny=>v001,
               p_kedvezmenyek=>:p_kedvezmenyek,
               p_dump_args#=>:p_dump_args#,
               p_szerz_azon=>:p_szerz_azon,
               p_ajanlat_url=>:p_ajanlat_url,
               p_szamolt_dijtetelek=>x021#,
               p_evesdij=>:p_evesdij,
               p_hibalista=>x026#,
               p_hiba_kod=>:p_hiba_kod,
               p_hiba_szov=>:p_hiba_szov);

  :x002#dijkod := v001.dijkod;
  :x002#dijfizmod := v001.dijfizmod;
  :x002#dijfizgyak := v001.dijfizgyak;
  :x002#szerkot := v001.szerkot;
  :x002#szerlejar := v001.szerlejar;
  :x002#kockezd := v001.kockezd;
  :x002#btkezd := v001.btkezd;
  :x002#halaszt_kockezd := v001.halaszt_kockezd;
  :x002#halaszt_dijfiz := v001.halaszt_dijfiz;
  :x002#szamlaszam := v001.szamlaszam;
  :x002#szamla_limit := v001.szamla_limit;
  :x002#evfordulo := v001.evfordulo;
  :x002#evfordulo_tipus := v001.evfordulo_tipus;
  :x002#e_komm_email := v001.e_komm_email;
  :x002#dijbekerot_ker := v001.dijbekerot_ker;
  :x002#ajanlati_evesdij := v001.ajanlati_evesdij;

  x021#ertek.DELETE;
  x021#nev.DELETE;
  x021#_idx := x021#.FIRST;
  WHILE x021#_idx IS NOT NULL LOOP
    x021#nev(x021#_idx) := x021#(x021#_idx).nev;
    x021#ertek(x021#_idx) := x021#(x021#_idx).ertek;
    x021#_idx := x021#.NEXT(x021#_idx);
  END LOOP;
  :x021#nev := x021#nev;
  :x021#ertek := x021#ertek;

  x026#szoveg.DELETE;
  x026#hibaszam.DELETE;
  x026#_idx := x026#.FIRST;
  WHILE x026#_idx IS NOT NULL LOOP
    x026#hibaszam(x026#_idx) := x026#(x026#_idx).hibaszam;
    x026#szoveg(x026#_idx) := x026#(x026#_idx).szoveg;
    x026#_idx := x026#.NEXT(x026#_idx);
  END LOOP;
  :x026#hibaszam := x026#hibaszam;
  :x027#szoveg := x026#szoveg;

END;
`,
	},
}

type testCase struct {
	Csv   string
	PlSql string
}

func (tc testCase) ParseCsv(t *testing.T, i int) []Function {
	functions, err := ParseCsv(strings.NewReader(tc.Csv), nil)
	if err != nil {
		t.Errorf("%d. error parsing csv: %v", i, err)
		t.FailNow()
	}
	return functions
}
