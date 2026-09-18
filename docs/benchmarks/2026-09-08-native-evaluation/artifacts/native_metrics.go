// Reuses the project's native evaluation calculators without changing their formulas.
package main
import("encoding/json";"os";"fmt";"github.com/Pototoooo/lorelattice/internal/application/service";"github.com/Pototoooo/lorelattice/internal/types")
type Input struct {Key string `json:"key"`; Answer string `json:"answer"`; Reference string `json:"reference"`; Retrieved []int `json:"retrieved"`; Relevant []int `json:"relevant"`}
type Output struct {Key string `json:"key"`; Metrics *types.MetricResult `json:"metrics"`}
func main(){var rows []Input;if err:=json.NewDecoder(os.Stdin).Decode(&rows);err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)};out:=[]Output{};for _,r:=range rows{m:=service.MetricList{};m.Append(&types.MetricInput{RetrievalGT:[][]int{r.Relevant},RetrievalIDs:r.Retrieved,GeneratedTexts:r.Answer,GeneratedGT:r.Reference});out=append(out,Output{r.Key,m.Avg()})};if len(os.Args)!=2{fmt.Fprintln(os.Stderr,"usage: native-evaluation-metrics OUTPUT.json < INPUT.json");os.Exit(2)};f,err:=os.Create(os.Args[1]);if err!=nil{panic(err)};defer f.Close();json.NewEncoder(f).Encode(out)}
