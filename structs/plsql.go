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

package structs

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/golang/glog"
)

// MaxTableSize is the maximum size of the array arguments
const MaxTableSize = 1000

//
// OracleArgument
//
var stringTypes = make(map[string]struct{}, 16)

func oracleVarTypeName(typ string) string {
	switch typ {
	case "BINARY_INTEGER", "PLS_INTEGER":
		return "int32"
	case "STRING", "VARCHAR2", "CHAR", "BINARY":
		return "string"
	case "INTEGER":
		return "int64"
	case "ROWID":
		return "[10]byte"
	case "REF CURSOR":
		return "gocilib.Cursor"
	case "TIMESTAMP":
		return "sqlhlp.NullTime"
	case "NUMBER":
		return "float64"
	case "DATE":
		return "sqlhlp.NullTime"
	case "BLOB":
		return "gocilib.LOB"
	case "CLOB":
		return "gocilib.LOB"
	case "BOOLEAN", "PL/SQL BOOLEAN":
		return "bool"
	default:
		glog.Fatalf("oracleVarTypeName: unknown variable type %q", typ)
	}
	return ""
}

/*
func oracleVarZero(typ string) string {
	switch typ {
	case "BINARY_INTEGER", "PLS_INTEGER":
		return "int32(0)"
	case "STRING", "VARCHAR2", "CHAR", "BINARY":
		return "\"\""
	case "INTEGER":
		return "int64(0)"
	case "ROWID":
		return "\"\""
	case "REF CURSOR":
		return "*gocilib.Cursor(nil)"
	//case "TIMESTAMP":
	//return "time.Time"
	case "NUMBER":
		return "float64(0)"
	case "DATE":
		return "time.Time{}"
	case "BLOB", "CLOB":
		return "*gocilib.LOB(nil)"
	case "BOOLEAN", "PL/SQL BOOLEAN":
		return "false"
	default:
		glog.Fatalf("oracleVarTypeName: unknown variable type %q", typ)
	}
	return ""
}
*/

// SavePlsqlBlock saves the plsql block definition into writer
func (fun Function) PlsqlBlock() (plsql, callFun string) {
	decls, pre, call, post, convIn, convOut, err := fun.prepareCall()
	if err != nil {
		glog.Fatalf("error preparing %s: %s", fun, err)
	}
	fn := strings.Replace(fun.Name(), ".", "__", -1)
	callBuf := bytes.NewBuffer(make([]byte, 0, 16384))
	fmt.Fprintf(callBuf, `func Call_%s(cx *gocilib.Connection, input %s) (output %s, err error) {
    if err = input.Check(); err != nil {
        return
    }
    params := make(map[string]driver.Value, %d)
    `, fn, fun.getStructName(false), fun.getStructName(true), len(fun.Args)+1)
	for _, line := range convIn {
		io.WriteString(callBuf, line+"\n")
	}
	i := strings.Index(call, fun.Name())
	j := i + strings.Index(call[i:], ")") + 1
	glog.V(2).Infof("i=%d j=%d call=\n%s", i, j, call)
	fmt.Fprintf(callBuf, "\nif true || DebugLevel > 0 { log.Printf(`calling %s\n\twith %%#v`, params) }"+`
	var stmt *gocilib.Statement
	if stmt, err = cx.NewPreparedStatement(%s); err != nil { return }
	defer stmt.Close()
    if err = stmt.BindExecute("", nil, params); err != nil { 
		err = fmt.Errorf("error executing %%q: with %%#v: %%v", %s, params, err)
		return
	}
	`, call[i:j], fun.getPlsqlConstName(), fun.getPlsqlConstName())
	for _, line := range convOut {
		io.WriteString(callBuf, line+"\n")
	}
	fmt.Fprintf(callBuf, `
        return
    }`)
	callFun = callBuf.String()

	plsBuf := callBuf
	plsBuf.Reset()
	if len(decls) > 0 {
		io.WriteString(plsBuf, "DECLARE\n")
		for _, line := range decls {
			fmt.Fprintf(plsBuf, "  %s\n", line)
		}
		plsBuf.Write([]byte{'\n'})
	}
	io.WriteString(plsBuf, "BEGIN\n")
	for _, line := range pre {
		fmt.Fprintf(plsBuf, "  %s\n", line)
	}
	fmt.Fprintf(plsBuf, "\n  %s;\n\n", call)
	for _, line := range post {
		fmt.Fprintf(plsBuf, "  %s\n", line)
	}
	io.WriteString(plsBuf, "\nEND;\n")
	plsql = plsBuf.String()
	return
}

func (fun Function) prepareCall() (decls, pre []string, call string, post []string, convIn, convOut []string, err error) {
	if fun.types == nil {
		glog.Infof("nil types of %s", fun)
		fun.types = make(map[string]string, 4)
	}
	tableTypes := make(map[string]string, 4)
	callArgs := make(map[string]string, 16)

	getTableType := func(absType string) string {
		typ, ok := tableTypes[absType]
		if !ok {
			typ = strings.Map(func(c rune) rune {
				switch c {
				case '(', ',':
					return '_'
				case ' ', ')':
					return -1
				default:
					return c
				}
			}, absType) + "_tab_typ"
			decls = append(decls, "TYPE "+typ+" IS TABLE OF "+absType+" INDEX BY BINARY_INTEGER;")
			tableTypes[absType] = typ
		}
		return typ
	}
	//fStructIn, fStructOut := fun.getStructName(false), fun.getStructName(true)
	var (
		vn, tmp, typ string
		ok           bool
	)
	decls = append(decls, "i1 PLS_INTEGER;", "i2 PLS_INTEGER;")
	convIn = append(convIn, "var v driver.Value\nvar x interface{}\n _, _ = v, x")

	var args []Argument
	if fun.Returns != nil {
		args = make([]Argument, 0, len(fun.Args)+1)
		for _, arg := range fun.Args {
			args = append(args, arg)
		}
		args = append(args, *fun.Returns)
	} else {
		args = fun.Args
	}
	for _, arg := range args {
		switch arg.Flavor {
		case FLAVOR_SIMPLE:
			name := capitalize(goName(arg.Name))
			convIn, convOut = arg.getConvSimple(convIn, convOut, fun.types, name, arg.Name, 0)

		case FLAVOR_RECORD:
			vn = getInnerVarName(fun.Name(), arg.Name)
			decls = append(decls, vn+" "+arg.TypeName+";")
			callArgs[arg.Name] = vn
			aname := capitalize(goName(arg.Name))
			if arg.IsOutput() {
				convOut = append(convOut, fmt.Sprintf(`
                    if output.%s == nil {
                        output.%s = new(%s)
                    }`, aname, aname, arg.goType(fun.types)[1:]))
			}
			for k, v := range arg.RecordOf {
				tmp = getParamName(fun.Name(), vn+"."+k)
				name := aname + "." + capitalize(goName(k))
				if arg.IsInput() {
					pre = append(pre, vn+"."+k+" := :"+tmp+";")
				}
				if arg.IsOutput() {
					post = append(post, ":"+tmp+" := "+vn+"."+k+";")
				}
				convIn, convOut = v.getConvRec(convIn, convOut, fun.types, name, tmp,
					0, &arg, k)
			}
		case FLAVOR_TABLE:
			if arg.Type == "REF CURSOR" {
				if arg.IsInput() {
					glog.Fatalf("cannot use IN cursor variables (%s)", arg)
				}
				name := capitalize(goName(arg.Name))
				convIn, convOut = arg.getConvSimple(convIn, convOut, fun.types, name, arg.Name, 0)
			} else {
				switch arg.TableOf.Flavor {
				case FLAVOR_SIMPLE: // like simple, but for the arg.TableOf
					typ = getTableType(arg.TableOf.AbsType)
					setvar := ""
					if arg.IsInput() {
						setvar = " := :" + arg.Name
					}
					decls = append(decls, arg.Name+" "+typ+setvar+";")

					vn = getInnerVarName(fun.Name(), arg.Name)
					callArgs[arg.Name] = vn
					decls = append(decls, vn+" "+arg.TypeName+";")
					if arg.IsInput() {
						pre = append(pre,
							vn+".DELETE;",
							"i1 := "+arg.Name+".FIRST;",
							"WHILE i1 IS NOT NULL LOOP",
							"  "+vn+"(i1) := "+arg.Name+"(i1);",
							"  i1 := "+arg.Name+".NEXT(i1);",
							"END LOOP;")
					}
					if arg.IsOutput() {
						post = append(post,
							arg.Name+".DELETE;",
							"i1 := "+vn+".FIRST;",
							"WHILE i1 IS NOT NULL LOOP",
							"  "+arg.Name+"(i1) := "+vn+"(i1);",
							"  i1 := "+vn+".NEXT(i1);",
							"END LOOP;",
							":"+arg.Name+" := "+arg.Name+";")
					}
					name := capitalize(goName(arg.Name))
					convIn, convOut = arg.TableOf.getConvSimple(convIn, convOut,
						fun.types, name, arg.Name, MaxTableSize)

				case FLAVOR_RECORD:
					vn = getInnerVarName(fun.Name(), arg.Name+"."+arg.TableOf.Name)
					callArgs[arg.Name] = vn
					decls = append(decls, vn+" "+arg.TypeName+";")

					//log.Printf("arg.Name=%q arg.TableOf.Name=%q arg.TableOf.RecordOf=%#v",
					//arg.Name, arg.TableOf.Name, arg.TableOf.RecordOf)
					aname := capitalize(goName(arg.Name))
					if arg.IsOutput() {
						convOut = append(convOut, fmt.Sprintf(`
                    if output.%s == nil {
                        output.%s = make([]%s, 0, %d)
                    }`, aname, aname, arg.TableOf.goType(fun.types), MaxTableSize))
					}
					/* // PLS-00110: a(z) 'P038.DELETE' hozzárendelt változó ilyen környezetben nem használható
					if arg.IsOutput() {
						// DELETE out tables
						for k := range arg.TableOf.RecordOf {
							post = append(post,
								":"+getParamName(fun.Name(), vn+"."+k)+".DELETE;")
						}
					}
					*/
					if !arg.IsInput() {
						pre = append(pre, vn+".DELETE;")
					}

					// declarations go first
					for k, v := range arg.TableOf.RecordOf {
						typ = getTableType(v.AbsType)
						decls = append(decls, getParamName(fun.Name(), vn+"."+k)+" "+typ+";")

						tmp = getParamName(fun.Name(), vn+"."+k)
						if arg.IsInput() {
							pre = append(pre, tmp+" := :"+tmp+";")
						} else {
							pre = append(pre, tmp+".DELETE;")
						}
					}

					// here comes the loops
					var idxvar string
					for k, v := range arg.TableOf.RecordOf {
						typ = getTableType(v.AbsType)

						tmp = getParamName(fun.Name(), vn+"."+k)

						if idxvar == "" {
							idxvar = getParamName(fun.Name(), vn+"."+k)
							if arg.IsInput() {
								pre = append(pre, "",
									"i1 := "+idxvar+".FIRST;",
									"WHILE i1 IS NOT NULL LOOP")
							}
							if arg.IsOutput() {
								post = append(post, "",
									"i1 := "+vn+".FIRST; i2 := 1;",
									"WHILE i1 IS NOT NULL LOOP")
							}
						}
						//name := aname + "." + capitalize(goName(k))

						convIn, convOut = v.getConvRec(
							convIn, convOut, fun.types, aname, tmp, MaxTableSize,
							arg.TableOf, k)

						if arg.IsInput() {
							pre = append(pre,
								"  "+vn+"(i1)."+k+" := "+tmp+"(i1);")
						}
						if arg.IsOutput() {
							post = append(post,
								"  "+tmp+"(i2) := "+vn+"(i1)."+k+";")
						}
					}
					if arg.IsInput() {
						pre = append(pre,
							"  i1 := "+idxvar+".NEXT(i1);",
							"END LOOP;")
					}
					if arg.IsOutput() {
						post = append(post,
							"  i1 := "+vn+".NEXT(i1); i2 := i2 + 1;",
							"END LOOP;")
						for k := range arg.TableOf.RecordOf {
							tmp = getParamName(fun.Name(), vn+"."+k)
							post = append(post, ":"+tmp+" := "+tmp+";")
						}
					}
				default:
					glog.Fatalf("%s/%s: only table of simple or record types are allowed (no table of table!)", fun.Name(), arg.Name)
				}
			}
		default:
			glog.Fatalf("unkown flavor %q", arg.Flavor)
		}
	}
	callb := bytes.NewBuffer(nil)
	if fun.Returns != nil {
		callb.WriteString(":ret := ")
	}
	glog.V(1).Infof("callArgs=%s", callArgs)
	callb.WriteString(fun.Name() + "(")
	for i, arg := range fun.Args {
		if i > 0 {
			callb.WriteString(",\n\t\t")
		}
		if vn, ok = callArgs[arg.Name]; !ok {
			vn = ":" + arg.Name
		}
		fmt.Fprintf(callb, "%s=>%s", arg.Name, vn)
	}
	callb.WriteString(")")
	call = callb.String()
	return
}

func notNullCheck(goType, name string) string {
	if strings.HasPrefix(goType, "Null") {
		return name + ".Valid"
	}
	if goType == "string" {
		return name + ` != "" `
	}
	return name + "!= nil"
}

func deRef(goType, name string) string {
	if strings.HasPrefix(goType, "Null") {
		return name + "." + goType[8:]
	}
	if goType[0] == '*' {
		return "*" + name
	}
	return name
}

func (arg Argument) getConvSimple(convIn, convOut []string, types map[string]string,
	name, paramName string, tableSize uint) ([]string, []string) {

	got := arg.goType(types)
	preconcept, preconcept2 := "", ""
	if strings.Count(name, ".") >= 1 {
		preconcept = "input." + name[:strings.LastIndex(name, ".")] + " != nil &&"
		preconcept2 = "if " + preconcept[:len(preconcept)-3] + " {"
	}

	if arg.IsOutput() {
		if arg.IsInput() {
			convIn = append(convIn,
				fmt.Sprintf("output.%s = input.%s", name, name))
			if got == "string" {
				convIn = append(convIn,
					fmt.Sprintf("v = gocilib.NewStringVar(input.%s, %d)", name, arg.CharLength))
			} else {
				convIn = append(convIn,
					fmt.Sprintf("v = &output.%s", name))
			}
		} else if got == "string" {
			convIn = append(convIn,
				fmt.Sprintf("v = gocilib.NewStringVar(\"\", %d)", arg.CharLength))
			convOut = append(convOut,
				fmt.Sprintf("if params[%q] != nil { output.%s = params[%q].(*gocilib.StringVar).String() }", paramName, name, paramName))
		} else {
			convIn = append(convIn,
				fmt.Sprintf("v = &output.%s", name))
		}
		pTyp := got
		if pTyp[0] == '*' {
			pTyp = pTyp[1:]
		}
		if tableSize == 0 {
			if strings.HasSuffix(got, "__cur") && arg.Type == "REF CURSOR" {
				pTyp = "*gocilib.Resultset"
			}
		}
		if arg.IsInput() {
			if tableSize == 0 {
				if preconcept2 != "" {
					convIn = append(convIn, preconcept2)
				}
				if preconcept2 != "" {
					convIn = append(convIn, "}")
				}
			} else {
				if preconcept2 != "" {
					convIn = append(convIn, preconcept2)
				}
				convIn = append(convIn,
					fmt.Sprintf(`v = input.%s`, name))
				if preconcept2 != "" {
					convIn = append(convIn, "}")
				}
			}
		}
	} else { // just and only input
		if tableSize == 0 {
			if preconcept2 != "" {
				convIn = append(convIn, preconcept2)
			}
			convIn = append(convIn,
				fmt.Sprintf(`v = input.%s`, name))
			if preconcept2 != "" {
				convIn = append(convIn, "} else { v = nil }")
			}
		} else {
			if preconcept2 != "" {
				convIn = append(convIn, preconcept2)
			}
			subGoType := got
			if subGoType[0] == '*' {
				subGoType = subGoType[1:]
			}
			/*
							convIn = append(convIn,
								fmt.Sprintf(`{
				                var a []%s
				                if len(input.%s) == 0 {
				                    a = make([]%s, 0)
				                } else {
				                    a = make([]%s, len(input.%s))
				                    for i, x := range input.%s {
				                        if `+notNullCheck(got, "x")+` { a[i] = `+deRef(got, "x")+` }
				                    }
				                }
				                v = a
				                }
				                    `,
									subGoType, name, subGoType, subGoType, name, name))
			*/
			convIn = append(convIn, "v = input."+name)
			if preconcept2 != "" {
				convIn = append(convIn, "}")
			}
		}
	}
	convIn = append(convIn,
		fmt.Sprintf("params[\"%s\"] = v\n", paramName))
	return convIn, convOut
}

func getOutConvTSwitch(name, pTyp string) string {
	parse := ""
	if strings.HasPrefix(pTyp, "int") {
		bits := "32"
		if len(pTyp) == 5 {
			bits = pTyp[3:5]
		}
		parse = "ParseInt(xi, 10, " + bits + ")"
	} else if strings.HasPrefix(pTyp, "float") {
		bits := pTyp[5:7]
		parse = "ParseFloat(xi, " + bits + ")"
	}
	if parse != "" {
		return fmt.Sprintf(`
			var y `+pTyp+`
			err = nil
			switch xi := x.(type) {
				case int: y = `+pTyp+`(xi)
				case int8: y = `+pTyp+`(xi)
				case int16: y = `+pTyp+`(xi)
				case int32: y = `+pTyp+`(xi)
				case int64: y = `+pTyp+`(xi)
				case float32: y = `+pTyp+`(xi)
				case float64: y = `+pTyp+`(xi)
				case string:
					//log.Printf("converting %%q to `+pTyp+`", xi)
					z, e := strconv.`+parse+`
					y, err = `+pTyp+`(z), e
				default:
					err = fmt.Errorf("out parameter %s is bad type: awaited %s, got %%T", x)
			}
			if err != nil {
				return
			}`, name, pTyp)
	}
	return fmt.Sprintf(`
				y, ok := x.(%s)
				if !ok {
					err = fmt.Errorf("out parameter %s is bad type: awaited %s, got %%T", x)
					return
				}`, pTyp, name, pTyp)
}

func (arg Argument) getConvRec(convIn, convOut []string,
	types map[string]string, name, paramName string, tableSize uint,
	parentArg *Argument, key string) ([]string, []string) {

	got := arg.goType(types)
	preconcept, preconcept2 := "", ""
	if strings.Count(name, ".") >= 1 && parentArg != nil {
		preconcept = "input." + name[:strings.LastIndex(name, ".")] + " != nil &&"
		preconcept2 = "if " + preconcept[:len(preconcept)-3] + " {"
	}
	if arg.IsOutput() {
		if arg.IsInput() {
			convIn = append(convIn,
				fmt.Sprintf("input.%s = output.%s", name, name))
		}
		convIn = append(convIn,
			fmt.Sprintf(`v = &output.%s`, name))
		pTyp := got
		if pTyp[0] == '*' {
			pTyp = pTyp[1:]
		}
		if arg.IsInput() {
			if tableSize == 0 {
				if preconcept2 != "" {
					convIn = append(convIn, preconcept2)
				}
				if preconcept2 != "" {
					convIn = append(convIn, "}")
				}
			} else {
				if preconcept2 != "" {
					convIn = append(convIn, preconcept2)
				}
				convIn = append(convIn,
					fmt.Sprintf(`v = input.%s`, name))
				if preconcept2 != "" {
					convIn = append(convIn, "}")
				}
			}
		}
	} else { // just and only input
		if tableSize == 0 {
			if preconcept2 != "" {
				convIn = append(convIn, preconcept2)
			}
			convIn = append(convIn,
				"v = input."+name)
			if preconcept2 != "" {
				convIn = append(convIn, "} else { v = nil }")
			}
		} else {
			if preconcept2 != "" {
				convIn = append(convIn, preconcept2)
			}
			subGoType := got
			if subGoType[0] == '*' {
				subGoType = subGoType[1:]
			}
			convIn = append(convIn,
				fmt.Sprintf(`{
                var a []%s
                if len(input.%s) == 0 {
                    a = make([]%s, 0)
                } else {
                    a = make([]%s, len(input.%s))
                    for i, x := range input.%s {
                   if x != nil && `+notNullCheck(got, "x."+capitalize(key))+` {
							a[i] = `+deRef(got, "x."+capitalize(key))+` }
                    }
                }
                v = a
                }
                    `,
					subGoType, name, subGoType, subGoType, name, name,
				))
			if preconcept2 != "" {
				convIn = append(convIn, "}")
			}
		}
	}
	convIn = append(convIn,
		fmt.Sprintf("params[\"%s\"] = v\n", paramName))
	return convIn, convOut
}

var varNames = make(map[string]map[string]string, 4)

func getVarName(funName, varName, prefix string) string {
	m, ok := varNames[funName]
	if !ok {
		m = make(map[string]string, 16)
		varNames[funName] = m
	}
	x, ok := m[varName]
	if !ok {
		length := len(m)
		if i := strings.LastIndex(varName, "."); i > 0 && i < len(varName)-1 {
			x = getVarName(funName, varName[:i], prefix) + "#" + varName[i+1:]
		}
		if x == "" || len(x) > 30 {
			x = fmt.Sprintf("%s%03d", prefix, length+1)
		}
		m[varName] = x
	}
	return x
}

func getInnerVarName(funName, varName string) string {
	return getVarName(funName, varName, "x")
}

func getParamName(funName, paramName string) string {
	return getVarName(funName, paramName, "x")
}
