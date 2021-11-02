/*
Copyright 2021 Cortex Labs, Inc.

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

package autoscaler

import (
	"fmt"
	"math"
	"time"
	"bufio"
	"encoding/csv"
	"os"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	math2 "github.com/cortexlabs/cortex/pkg/lib/math"
	"github.com/cortexlabs/cortex/pkg/lib/strings"
	time2 "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	kapps "k8s.io/api/apps/v1"
	sstrings "strings"
	"strconv"
)

// GetInFlightFunc is the function signature used by the autoscaler to retrieve
// the number of in-flight requests / messages
type GetInFlightFunc func(apiName string, window time.Duration) (*float64, error)

type recommendations map[time.Time]int32

func (recs recommendations) add(rec int32) {
	recs[time.Now()] = rec
}

func (recs recommendations) deleteOlderThan(period time.Duration) {
	for t := range recs {
		if time.Since(t) > period {
			delete(recs, t)
		}
	}
}

// Returns nil if no recommendations in the period
func (recs recommendations) maxSince(period time.Duration) *int32 {
	max := int32(math.MinInt32)
	foundRecommendation := false

	for t, rec := range recs {
		if time.Since(t) <= period && rec > max {
			max = rec
			foundRecommendation = true
		}
	}

	if !foundRecommendation {
		return nil
	}

	return &max
}

// Returns nil if no recommendations in the period
func (recs recommendations) minSince(period time.Duration) *int32 {
	min := int32(math.MaxInt32)
	foundRecommendation := false

	for t, rec := range recs {
		if time.Since(t) <= period && rec < min {
			min = rec
			foundRecommendation = true
		}
	}

	if !foundRecommendation {
		return nil
	}

	return &min
}

// AutoscaleFn returns the autoscaler function
func AutoscaleFn(initialDeployment *kapps.Deployment, apiSpec *spec.API, getInFlightFn GetInFlightFunc) (func() error, error) {
	autoscalingSpec, err := userconfig.AutoscalingFromAnnotations(initialDeployment)
	if err != nil {
		return nil, err
	}

	apiName := apiSpec.Name
	currentReplicas := *initialDeployment.Spec.Replicas

	apiLogger, err := operator.GetRealtimeAPILoggerFromSpec(apiSpec)
	if err != nil {
		return nil, err
	}


	apiLogger.Debugf("%s wja300 autoscaler init", apiName)

	var startTime time.Time
	recs := make(recommendations)

	return func() error {
		if startTime.IsZero() {
			startTime = time.Now()
		}

		avgInFlight, err := getInFlightFn(apiName, autoscalingSpec.Window)
		if err != nil {
			return err
		}
		if avgInFlight == nil {
			apiLogger.Debugf("%s autoscaler tick: metrics not available yet", apiName)
			return nil
		}

		rawRecommendation := *avgInFlight / *autoscalingSpec.TargetReplicaConcurrency
		recommendation := int32(math.Ceil(rawRecommendation))

		if rawRecommendation < float64(currentReplicas) && rawRecommendation > float64(currentReplicas)*(1-autoscalingSpec.DownscaleTolerance) {
			recommendation = currentReplicas
		}

		if rawRecommendation > float64(currentReplicas) && rawRecommendation < float64(currentReplicas)*(1+autoscalingSpec.UpscaleTolerance) {
			recommendation = currentReplicas
		}

		// always allow subtraction of 1
		downscaleFactorFloor := math2.MinInt32(currentReplicas-1, int32(math.Ceil(float64(currentReplicas)*autoscalingSpec.MaxDownscaleFactor)))
		if recommendation < downscaleFactorFloor {
			recommendation = downscaleFactorFloor
		}

//		dir, err := os.Getwd()
//		if err != nil {
//			fmt.Print("wja300 getwd error:\n", err)
//		}
//		fmt.Print("wja300 current directory : ", dir)

		// 파일 오픈
		file, err := os.Open("./tweet_load_predict_5min_10-16.csv")
		if err != nil {	
			fmt.Print("wja300 os open error:\n", err)
		}
		// csv reader 생성
		rdr := csv.NewReader(bufio.NewReader(file))
		// csv 내용 모두 읽기
		rows, err := rdr.ReadAll()
		if err != nil {	
			fmt.Print("wja300 ReadAll error:\n", err)
		}
		file.Close()

		// 파일 오픈
		file_spec, err := os.Open("./spec.csv")
		if err != nil {	
			fmt.Print("wja300 spec.csv open error:\n", err)
		}
		// csv reader 생성
		rdr_spec := csv.NewReader(bufio.NewReader(file_spec))
		// csv 내용 모두 읽기
		rows_spec, err := rdr_spec.ReadAll()
		if err != nil {	
			fmt.Print("wja300 rows_spec ReadAll error:\n", err)
		}
		file_spec.Close()



		//fmt.Print("wja300:\n", rows)
		

		// always allow addition of 1
		upscaleFactorCeil := math2.MaxInt32(currentReplicas+1, int32(math.Ceil(float64(currentReplicas)*autoscalingSpec.MaxUpscaleFactor)))
		if recommendation > upscaleFactorCeil {
			recommendation = upscaleFactorCeil
		}

		if recommendation < 1 {
			recommendation = 1
		}

		if recommendation < autoscalingSpec.MinReplicas {
			recommendation = autoscalingSpec.MinReplicas
		}

		if recommendation > autoscalingSpec.MaxReplicas {
			recommendation = autoscalingSpec.MaxReplicas
		}

		// Rule of thumb: any modifications that don't consider historical recommendations should be performed before
		// recording the recommendation, any modifications that use historical recommendations should be performed after
		recs.add(recommendation)

		// This is just for garbage collection
		recs.deleteOlderThan(time2.MaxDuration(autoscalingSpec.DownscaleStabilizationPeriod, autoscalingSpec.UpscaleStabilizationPeriod))

		request := recommendation
		var downscaleStabilizationFloor *int32
		var upscaleStabilizationCeil *int32

		if request < currentReplicas {
			downscaleStabilizationFloor = recs.maxSince(autoscalingSpec.DownscaleStabilizationPeriod)
			if time.Since(startTime) < autoscalingSpec.DownscaleStabilizationPeriod {
				request = currentReplicas
			} else if downscaleStabilizationFloor != nil && request < *downscaleStabilizationFloor {
				request = *downscaleStabilizationFloor
			}
		}
		if request > currentReplicas {
			upscaleStabilizationCeil = recs.minSince(autoscalingSpec.UpscaleStabilizationPeriod)
			if time.Since(startTime) < autoscalingSpec.UpscaleStabilizationPeriod {
				request = currentReplicas
			} else if upscaleStabilizationCeil != nil && request > *upscaleStabilizationCeil {
				request = *upscaleStabilizationCeil
			}
		}

		apiLogger.Debugw(fmt.Sprintf("%s autoscaler tick", apiName),
			"autoscaling", map[string]interface{}{
				"avg_in_flight":                  strings.Round(*avgInFlight, 2, 0),
				"target_replica_concurrency":     strings.Float64(*autoscalingSpec.TargetReplicaConcurrency),
				"raw_recommendation":             strings.Round(rawRecommendation, 2, 0),
				"current_replicas":               currentReplicas,
				"downscale_tolerance":            strings.Float64(autoscalingSpec.DownscaleTolerance),
				"upscale_tolerance":              strings.Float64(autoscalingSpec.UpscaleTolerance),
				"max_downscale_factor":           strings.Float64(autoscalingSpec.MaxDownscaleFactor),
				"downscale_factor_floor":         downscaleFactorFloor,
				"max_upscale_factor":             strings.Float64(autoscalingSpec.MaxUpscaleFactor),
				"upscale_factor_ceil":            upscaleFactorCeil,
				"min_replicas":                   autoscalingSpec.MinReplicas,
				"max_replicas":                   autoscalingSpec.MaxReplicas,
				"recommendation":                 recommendation,
				"downscale_stabilization_period": autoscalingSpec.DownscaleStabilizationPeriod,
				"downscale_stabilization_floor":  strings.ObjFlatNoQuotes(downscaleStabilizationFloor),
				"upscale_stabilization_period":   autoscalingSpec.UpscaleStabilizationPeriod,
				"upscale_stabilization_ceil":     strings.ObjFlatNoQuotes(upscaleStabilizationCeil),
				"request":                        request,
				"Window" :                        autoscalingSpec.Window,
				"Period" :			  time.Since(startTime),
				"Period_second" :		  int32(time.Since(startTime)/1000000000),
				"Rows" :                          rows[1],
			},
		)

		num_rows := 0
		
		// 행,열 읽기
    		for i, _ := range rows {
		num_rows = i
    		}

//		num_rows_spec := 0
		
		// 행,열 읽기
  //  		for i, _ := range rows_spec {
//		num_rows_spec = i
  //  		}


	 	period := time.Since(startTime) // \"period\":4992215247666
		period_second := int32(time.Since(startTime)/1000000000) // \"period_second\":4992 // 499 get (by / 10000000000) 499*10 // how to get "2" how to get "4990"
		// \"Period\":5412294863623,\"Period_second\":5412 // 541 get 5410 // if 5422 542 get 5420 // {\"Period\":130027293946,\"Period_second\":130 // 130 - 130 = 0)

		period_second_mul := int32(time.Since(startTime)/10000000000) // 499 // 13
		period_second_mul = period_second_mul*10 // 4990 // 130
		time_correction := period_second - period_second_mul // 4992 - 4990 = 2 // 130 - 130 = 0 // if \"Period\":7212738760866,\"Period_second\":7212


		row_count := period_second / 60
		timeout := 360
		num_reqs := 0.0
		conv_reqs := 0.0
		reqs_base := 0.0
		workload_ratio := 0.0
		workload_instance_ratio := 0.0
		amplification := 0.0

		fmt.Print(" time_correction : ", time_correction)
		fmt.Print(" period_second - time_correction : ", period_second - time_correction)
		fmt.Print(" (period_second - time_correction) % 60 : ", (period_second -time_correction) % 60)
                fmt.Print(" wja300 num_rows : ", num_rows)
		fmt.Print(" wja300 row_count : ", row_count)
		fmt.Print(" wja300 timeout : ", timeout)

		if (row_count  > int32(timeout) || row_count +4 >= int32(num_rows)){
			// nothing done
		} else if (sstrings.Contains(apiName, "resnet50") && sstrings.Contains(apiName, "i1") && sstrings.Contains(apiName, "37") ){  // alg3.7 i1 - resnet50
				if ((period_second - time_correction) % 60 <= 9) { // every 1 minutes
					num_reqs, _ = strconv.ParseFloat(rows[row_count+4][1], 64)
					// weight conversion (alg3.7) num_reqs -> converting
					for i, _ := range rows_spec {
						if (rows_spec[i][0] == "i1R"){
							reqs_base, _ = strconv.ParseFloat(rows_spec[i][1], 64) // rows_sepc[i][1] = i1R's reqs_base
							fmt.Print(" wja300 R reqs_base : ", reqs_base)
						}
						if (rows_spec[i][0] == "ratio"){
							workload_ratio, _  = strconv.ParseFloat(rows_spec[i][1], 64) // rows_spec[i][1]:R'ratio, [i][2]:B, [i][3]:G, [i][4]:Y, [i][5]:S
							fmt.Print(" wja300 workload_ratio : ", workload_ratio)
						}
						if (rows_spec[i][0] == "alg37i1"){
							workload_instance_ratio, _  = strconv.ParseFloat(rows_spec[i][1], 64) // rows_spec[i][1]:R's I ratio, [i][2]:B's I ratio, [i][3]:G's I, [i][4]:Y's I, [i][5]:S's I
							fmt.Print(" wja300 workload_instance_ratio : ", workload_instance_ratio)
						}
						if (rows_spec[i][0] == "amplification"){
							amplification, _ = strconv.ParseFloat(rows_spec[i][1], 64) 
							fmt.Print(" wja300 amplification : ", amplification)
						}
					}
					num_reqs = num_reqs*amplification
					num_reqs = num_reqs*workload_ratio/100*workload_instance_ratio/100 // alg3.7 i1 - resnet50 converting
					conv_reqs = float64(num_reqs) / reqs_base // i1 - resnet50
					if(request > int32(math.Ceil(conv_reqs))){
						// nothing done
					} else {
						request = int32(math.Ceil(conv_reqs))
					}
					fmt.Print(" wja300 num_reqs : ", num_reqs)
					fmt.Print(" wja300 conv_reqs : ", conv_reqs)
					fmt.Print(" wja300 request : ", request)
				}
		} else if (sstrings.Contains(apiName, "sentiment") && sstrings.Contains(apiName, "i1") && sstrings.Contains(apiName, "37") ){  // alg3.7 i1 - bert
				if ((period_second - time_correction) % 60 <= 9) { // every 1 minutes
					num_reqs, _ = strconv.ParseFloat(rows[row_count+4][1], 64)
					// weight conversion (alg3.7) num_reqs -> converting
					for i, _ := range rows_spec {
						if (rows_spec[i][0] == "i1B"){
							reqs_base, _ = strconv.ParseFloat(rows_spec[i][1], 64) // rows_sepc[i][1] = i1R's reqs_base
							fmt.Print(" wja300 B reqs_base : ", reqs_base)
						}
						if (rows_spec[i][0] == "ratio"){
							workload_ratio, _  = strconv.ParseFloat(rows_spec[i][2], 64) // rows_spec[i][1]:R'ratio, [i][2]:B, [i][3]:G, [i][4]:Y, [i][5]:S
							fmt.Print(" wja300 workload_ratio : ", workload_ratio)
						}
						if (rows_spec[i][0] == "alg37i1"){
							workload_instance_ratio, _  = strconv.ParseFloat(rows_spec[i][2], 64) // rows_spec[i][1]:R's I ratio, [i][2]:B's I ratio, [i][3]:G's I, [i][4]:Y's I, [i][5]:S's I
							fmt.Print(" wja300 workload_instance_ratio : ", workload_instance_ratio)
						}
						if (rows_spec[i][0] == "amplification"){
							amplification, _ = strconv.ParseFloat(rows_spec[i][1], 64) 
							fmt.Print(" wja300 amplification : ", amplification)
						}
					}
					num_reqs = num_reqs*amplification
					num_reqs = num_reqs*workload_ratio/100*workload_instance_ratio/100 // alg3.7 i1 - resnet50 converting
					conv_reqs = float64(num_reqs) / reqs_base // i1 - resnet50
					if(request > int32(math.Ceil(conv_reqs))){
						// nothing done
					} else {
						request = int32(math.Ceil(conv_reqs))
					}
					fmt.Print(" wja300 num_reqs : ", num_reqs)
					fmt.Print(" wja300 conv_reqs : ", conv_reqs)
					fmt.Print(" wja300 request : ", request)
				}
		} else if (sstrings.Contains(apiName, "text") && sstrings.Contains(apiName, "i1") && sstrings.Contains(apiName, "37") ){  // alg3.7 i1 - gpt
				if ((period_second - time_correction) % 60 <= 9) { // every 1 minutes
					num_reqs, _ = strconv.ParseFloat(rows[row_count+4][1], 64)
					// weight conversion (alg3.7) num_reqs -> converting
					for i, _ := range rows_spec {
						if (rows_spec[i][0] == "i1G"){
							reqs_base, _ = strconv.ParseFloat(rows_spec[i][1], 64) // rows_sepc[i][1] = i1R's reqs_base
							fmt.Print(" wja300 G reqs_base : ", reqs_base)
						}
						if (rows_spec[i][0] == "ratio"){
							workload_ratio, _  = strconv.ParseFloat(rows_spec[i][3], 64) // rows_spec[i][1]:R'ratio, [i][2]:B, [i][3]:G, [i][4]:Y, [i][5]:S
							fmt.Print(" wja300 workload_ratio : ", workload_ratio)
						}
						if (rows_spec[i][0] == "alg37i1"){
							workload_instance_ratio, _  = strconv.ParseFloat(rows_spec[i][3], 64) // rows_spec[i][1]:R's I ratio, [i][2]:B's I ratio, [i][3]:G's I, [i][4]:Y's I, [i][5]:S's I
							fmt.Print(" wja300 workload_instance_ratio : ", workload_instance_ratio)
						}
						if (rows_spec[i][0] == "amplification"){
							amplification, _ = strconv.ParseFloat(rows_spec[i][1], 64) 
							fmt.Print(" wja300 amplification : ", amplification)
						}
					}
					num_reqs = num_reqs*amplification
					num_reqs = num_reqs*workload_ratio/100*workload_instance_ratio/100 // alg3.7 i1 - resnet50 converting
					conv_reqs = float64(num_reqs) / reqs_base // i1 - resnet50
					if(request > int32(math.Ceil(conv_reqs))){
						// nothing done
					} else {
						request = int32(math.Ceil(conv_reqs))
					}
					fmt.Print(" wja300 num_reqs : ", num_reqs)
					fmt.Print(" wja300 conv_reqs : ", conv_reqs)
					fmt.Print(" wja300 request : ", request)
				}
		} else if (sstrings.Contains(apiName, "sound") && sstrings.Contains(apiName, "i1") && sstrings.Contains(apiName, "37") ){  // alg3.7 i1 - yamnet
				if ((period_second - time_correction) % 60 <= 9) { // every 1 minutes
					num_reqs, _ = strconv.ParseFloat(rows[row_count+4][1], 64)
					// weight conversion (alg3.7) num_reqs -> converting
					for i, _ := range rows_spec {
						if (rows_spec[i][0] == "i1Y"){
							reqs_base, _ = strconv.ParseFloat(rows_spec[i][1], 64) // rows_sepc[i][1] = i1R's reqs_base
							fmt.Print(" wja300 Y reqs_base : ", reqs_base)
						}
						if (rows_spec[i][0] == "ratio"){
							workload_ratio, _  = strconv.ParseFloat(rows_spec[i][4], 64) // rows_spec[i][1]:R'ratio, [i][2]:B, [i][3]:G, [i][4]:Y, [i][5]:S
							fmt.Print(" wja300 workload_ratio : ", workload_ratio)
						}
						if (rows_spec[i][0] == "alg37i1"){
							workload_instance_ratio, _  = strconv.ParseFloat(rows_spec[i][4], 64) // rows_spec[i][1]:R's I ratio, [i][2]:B's I ratio, [i][3]:G's I, [i][4]:Y's I, [i][5]:S's I
							fmt.Print(" wja300 workload_instance_ratio : ", workload_instance_ratio)
						}
						if (rows_spec[i][0] == "amplification"){
							amplification, _ = strconv.ParseFloat(rows_spec[i][1], 64) 
							fmt.Print(" wja300 amplification : ", amplification)
						}
					}
					num_reqs = num_reqs*amplification
					num_reqs = num_reqs*workload_ratio/100*workload_instance_ratio/100 // alg3.7 i1 - resnet50 converting
					conv_reqs = float64(num_reqs) / reqs_base // i1 - resnet50
					if(request > int32(math.Ceil(conv_reqs))){
						// nothing done
					} else {
						request = int32(math.Ceil(conv_reqs))
					}
					fmt.Print(" wja300 num_reqs : ", num_reqs)
					fmt.Print(" wja300 conv_reqs : ", conv_reqs)
					fmt.Print(" wja300 request : ", request)
				}
		} else if (sstrings.Contains(apiName, "inception") && sstrings.Contains(apiName, "i1") && sstrings.Contains(apiName, "37") ){  // alg3.7 i1 - inception
				if ((period_second - time_correction) % 60 <= 9) { // every 1 minutes
					num_reqs, _ = strconv.ParseFloat(rows[row_count+4][1], 64)
					// weight conversion (alg3.7) num_reqs -> converting
					for i, _ := range rows_spec {
						if (rows_spec[i][0] == "i1S"){
							reqs_base, _ = strconv.ParseFloat(rows_spec[i][1], 64) // rows_sepc[i][1] = i1R's reqs_base
							fmt.Print(" wja300 S reqs_base : ", reqs_base)
						}
						if (rows_spec[i][0] == "ratio"){
							workload_ratio, _  = strconv.ParseFloat(rows_spec[i][5], 64) // rows_spec[i][1]:R'ratio, [i][2]:B, [i][3]:G, [i][4]:Y, [i][5]:S
							fmt.Print(" wja300 workload_ratio : ", workload_ratio)
						}
						if (rows_spec[i][0] == "alg37i1"){
							workload_instance_ratio, _  = strconv.ParseFloat(rows_spec[i][5], 64) // rows_spec[i][1]:R's I ratio, [i][2]:B's I ratio, [i][3]:G's I, [i][4]:Y's I, [i][5]:S's I
							fmt.Print(" wja300 workload_instance_ratio : ", workload_instance_ratio)
						}
						if (rows_spec[i][0] == "amplification"){
							amplification, _ = strconv.ParseFloat(rows_spec[i][1], 64) 
							fmt.Print(" wja300 amplification : ", amplification)
						}
					}
					num_reqs = num_reqs*amplification
					num_reqs = num_reqs*workload_ratio/100*workload_instance_ratio/100 // alg3.7 i1 - resnet50 converting
					conv_reqs = float64(num_reqs) / reqs_base // i1 - resnet50
					if(request > int32(math.Ceil(conv_reqs))){
						// nothing done
					} else {
						request = int32(math.Ceil(conv_reqs))
					}
					fmt.Print(" wja300 num_reqs : ", num_reqs)
					fmt.Print(" wja300 conv_reqs : ", conv_reqs)
					fmt.Print(" wja300 request : ", request)
				}
		} else if (sstrings.Contains(apiName, "resnet50") && sstrings.Contains(apiName, "p2") && sstrings.Contains(apiName, "37") ){  // alg3.7 p2 - resnet50
				if ((period_second - time_correction) % 60 <= 9) { // every 1 minutes
					num_reqs, _ = strconv.ParseFloat(rows[row_count+4][1], 64)
					// weight conversion (alg3.7) num_reqs -> converting
					for i, _ := range rows_spec {
						if (rows_spec[i][0] == "p2R"){
							reqs_base, _ = strconv.ParseFloat(rows_spec[i][1], 64) // rows_sepc[i][1] = i1R's reqs_base
							fmt.Print(" wja300 R reqs_base : ", reqs_base)
						}
						if (rows_spec[i][0] == "ratio"){
							workload_ratio, _  = strconv.ParseFloat(rows_spec[i][1], 64) // rows_spec[i][1]:R'ratio, [i][2]:B, [i][3]:G, [i][4]:Y, [i][5]:S
							fmt.Print(" wja300 workload_ratio : ", workload_ratio)
						}
						if (rows_spec[i][0] == "alg37p2"){
							workload_instance_ratio, _  = strconv.ParseFloat(rows_spec[i][1], 64) // rows_spec[i][1]:R's I ratio, [i][2]:B's I ratio, [i][3]:G's I, [i][4]:Y's I, [i][5]:S's I
							fmt.Print(" wja300 workload_instance_ratio : ", workload_instance_ratio)
						}
						if (rows_spec[i][0] == "amplification"){
							amplification, _ = strconv.ParseFloat(rows_spec[i][1], 64) 
							fmt.Print(" wja300 amplification : ", amplification)
						}
					}
					num_reqs = num_reqs*amplification
					num_reqs = num_reqs*workload_ratio/100*workload_instance_ratio/100 // alg3.7 i1 - resnet50 converting
					conv_reqs = float64(num_reqs) / reqs_base // i1 - resnet50
					if(request > int32(math.Ceil(conv_reqs))){
						// nothing done
					} else {
						request = int32(math.Ceil(conv_reqs))
					}
					fmt.Print(" wja300 num_reqs : ", num_reqs)
					fmt.Print(" wja300 conv_reqs : ", conv_reqs)
					fmt.Print(" wja300 request : ", request)
				}
		} else if (sstrings.Contains(apiName, "sentiment") && sstrings.Contains(apiName, "p2") && sstrings.Contains(apiName, "37") ){  // alg3.7 p2 - bert
				if ((period_second - time_correction) % 60 <= 9) { // every 1 minutes
					num_reqs, _ = strconv.ParseFloat(rows[row_count+4][1], 64)
					// weight conversion (alg3.7) num_reqs -> converting
					for i, _ := range rows_spec {
						if (rows_spec[i][0] == "p2B"){
							reqs_base, _ = strconv.ParseFloat(rows_spec[i][1], 64) // rows_sepc[i][1] = i1R's reqs_base
							fmt.Print(" wja300 B reqs_base : ", reqs_base)
						}
						if (rows_spec[i][0] == "ratio"){
							workload_ratio, _  = strconv.ParseFloat(rows_spec[i][2], 64) // rows_spec[i][1]:R'ratio, [i][2]:B, [i][3]:G, [i][4]:Y, [i][5]:S
							fmt.Print(" wja300 workload_ratio : ", workload_ratio)
						}
						if (rows_spec[i][0] == "alg37p2"){
							workload_instance_ratio, _  = strconv.ParseFloat(rows_spec[i][2], 64) // rows_spec[i][1]:R's I ratio, [i][2]:B's I ratio, [i][3]:G's I, [i][4]:Y's I, [i][5]:S's I
							fmt.Print(" wja300 workload_instance_ratio : ", workload_instance_ratio)
						}
						if (rows_spec[i][0] == "amplification"){
							amplification, _ = strconv.ParseFloat(rows_spec[i][1], 64) 
							fmt.Print(" wja300 amplification : ", amplification)
						}
					}
					num_reqs = num_reqs*amplification
					num_reqs = num_reqs*workload_ratio/100*workload_instance_ratio/100 // alg3.7 i1 - resnet50 converting
					conv_reqs = float64(num_reqs) / reqs_base // i1 - resnet50
					if(request > int32(math.Ceil(conv_reqs))){
						// nothing done
					} else {
						request = int32(math.Ceil(conv_reqs))
					}
					fmt.Print(" wja300 num_reqs : ", num_reqs)
					fmt.Print(" wja300 conv_reqs : ", conv_reqs)
					fmt.Print(" wja300 request : ", request)
				}
		} else if (sstrings.Contains(apiName, "text") && sstrings.Contains(apiName, "p2") && sstrings.Contains(apiName, "37") ){  // alg3.7 p2 - gpt
				if ((period_second - time_correction) % 60 <= 9) { // every 1 minutes
					num_reqs, _ = strconv.ParseFloat(rows[row_count+4][1], 64)
					// weight conversion (alg3.7) num_reqs -> converting
					for i, _ := range rows_spec {
						if (rows_spec[i][0] == "p2G"){
							reqs_base, _ = strconv.ParseFloat(rows_spec[i][1], 64) // rows_sepc[i][1] = i1R's reqs_base
								fmt.Print(" wja300 G reqs_base : ", reqs_base)
						}
						if (rows_spec[i][0] == "ratio"){
							workload_ratio, _  = strconv.ParseFloat(rows_spec[i][3], 64) // rows_spec[i][1]:R'ratio, [i][2]:B, [i][3]:G, [i][4]:Y, [i][5]:S
							fmt.Print(" wja300 workload_ratio : ", workload_ratio)
						}
						if (rows_spec[i][0] == "alg37p2"){
							workload_instance_ratio, _  = strconv.ParseFloat(rows_spec[i][3], 64) // rows_spec[i][1]:R's I ratio, [i][2]:B's I ratio, [i][3]:G's I, [i][4]:Y's I, [i][5]:S's I
							fmt.Print(" wja300 workload_instance_ratio : ", workload_instance_ratio)
						}
						if (rows_spec[i][0] == "amplification"){
							amplification, _ = strconv.ParseFloat(rows_spec[i][1], 64) 
							fmt.Print(" wja300 amplification : ", amplification)
						}
					}
					num_reqs = num_reqs*amplification
					num_reqs = num_reqs*workload_ratio/100*workload_instance_ratio/100 // alg3.7 i1 - resnet50 converting
					conv_reqs = float64(num_reqs) / reqs_base // i1 - resnet50
					if(request > int32(math.Ceil(conv_reqs))){
						// nothing done
					} else {
						request = int32(math.Ceil(conv_reqs))
					}
					fmt.Print(" wja300 num_reqs : ", num_reqs)
					fmt.Print(" wja300 conv_reqs : ", conv_reqs)
					fmt.Print(" wja300 request : ", request)
				}
		} else if (sstrings.Contains(apiName, "sound") && sstrings.Contains(apiName, "p2") && sstrings.Contains(apiName, "37") ){  // alg3.7 p2 - yamnet
				if ((period_second - time_correction) % 60 <= 9) { // every 1 minutes
					num_reqs, _ = strconv.ParseFloat(rows[row_count+4][1], 64)
					// weight conversion (alg3.7) num_reqs -> converting
					for i, _ := range rows_spec {
						if (rows_spec[i][0] == "p2Y"){
							reqs_base, _ = strconv.ParseFloat(rows_spec[i][1], 64) // rows_sepc[i][1] = i1R's reqs_base
							fmt.Print(" wja300 Y reqs_base : ", reqs_base)
						}
						if (rows_spec[i][0] == "ratio"){
							workload_ratio, _  = strconv.ParseFloat(rows_spec[i][4], 64) // rows_spec[i][1]:R'ratio, [i][2]:B, [i][3]:G, [i][4]:Y, [i][5]:S
							fmt.Print(" wja300 workload_ratio : ", workload_ratio)
						}
						if (rows_spec[i][0] == "alg37p2"){
							workload_instance_ratio, _  = strconv.ParseFloat(rows_spec[i][4], 64) // rows_spec[i][1]:R's I ratio, [i][2]:B's I ratio, [i][3]:G's I, [i][4]:Y's I, [i][5]:S's I
							fmt.Print(" wja300 workload_instance_ratio : ", workload_instance_ratio)
						}
						if (rows_spec[i][0] == "amplification"){
							amplification, _ = strconv.ParseFloat(rows_spec[i][1], 64) 
							fmt.Print(" wja300 amplification : ", amplification)
						}
					}
					num_reqs = num_reqs*amplification
					num_reqs = num_reqs*workload_ratio/100*workload_instance_ratio/100 // alg3.7 i1 - resnet50 converting
					conv_reqs = float64(num_reqs) / reqs_base // i1 - resnet50
					if(request > int32(math.Ceil(conv_reqs))){
						// nothing done
					} else {
						request = int32(math.Ceil(conv_reqs))
					}
					fmt.Print(" wja300 num_reqs : ", num_reqs)
					fmt.Print(" wja300 conv_reqs : ", conv_reqs)
					fmt.Print(" wja300 request : ", request)
				}
		} else if (sstrings.Contains(apiName, "inception") && sstrings.Contains(apiName, "p2") && sstrings.Contains(apiName, "37") ){  // alg3.7 p2 - inception
				if ((period_second - time_correction) % 60 <= 9) { // every 1 minutes
					num_reqs, _ = strconv.ParseFloat(rows[row_count+4][1], 64)
					// weight conversion (alg3.7) num_reqs -> converting
					for i, _ := range rows_spec {
						if (rows_spec[i][0] == "p2S"){
							reqs_base, _ = strconv.ParseFloat(rows_spec[i][1], 64) // rows_sepc[i][1] = i1R's reqs_base
							fmt.Print(" wja300 S reqs_base : ", reqs_base)
						}
						if (rows_spec[i][0] == "ratio"){
							workload_ratio, _  = strconv.ParseFloat(rows_spec[i][5], 64) // rows_spec[i][1]:R'ratio, [i][2]:B, [i][3]:G, [i][4]:Y, [i][5]:S
							fmt.Print(" wja300 workload_ratio : ", workload_ratio)
						}
						if (rows_spec[i][0] == "alg37p2"){
							workload_instance_ratio, _  = strconv.ParseFloat(rows_spec[i][5], 64) // rows_spec[i][1]:R's I ratio, [i][2]:B's I ratio, [i][3]:G's I, [i][4]:Y's I, [i][5]:S's I
							fmt.Print(" wja300 workload_instance_ratio : ", workload_instance_ratio)
						}
						if (rows_spec[i][0] == "amplification"){
							amplification, _ = strconv.ParseFloat(rows_spec[i][1], 64) 
							fmt.Print(" wja300 amplification : ", amplification)
						}
					}
					num_reqs = num_reqs*amplification
					num_reqs = num_reqs*workload_ratio/100*workload_instance_ratio/100 // alg3.7 i1 - resnet50 converting
					conv_reqs = float64(num_reqs) / reqs_base // i1 - resnet50
					if(request > int32(math.Ceil(conv_reqs))){
						// nothing done
					} else {
						request = int32(math.Ceil(conv_reqs))
					}
					fmt.Print(" wja300 num_reqs : ", num_reqs)
					fmt.Print(" wja300 conv_reqs : ", conv_reqs)
					fmt.Print(" wja300 request : ", request)
				}
		} else if (sstrings.Contains(apiName, "resnet50") && sstrings.Contains(apiName, "p3") && sstrings.Contains(apiName, "37") ){  // alg3.7 p3 - resnet50
				if ((period_second - time_correction) % 60 <= 9) { // every 1 minutes
					num_reqs, _ = strconv.ParseFloat(rows[row_count+4][1], 64)
					// weight conversion (alg3.7) num_reqs -> converting
					for i, _ := range rows_spec {
						if (rows_spec[i][0] == "p3R"){
							reqs_base, _ = strconv.ParseFloat(rows_spec[i][1], 64) // rows_sepc[i][1] = i1R's reqs_base
							fmt.Print(" wja300 R reqs_base : ", reqs_base)
						}
						if (rows_spec[i][0] == "ratio"){
							workload_ratio, _  = strconv.ParseFloat(rows_spec[i][1], 64) // rows_spec[i][1]:R'ratio, [i][2]:B, [i][3]:G, [i][4]:Y, [i][5]:S
							fmt.Print(" wja300 workload_ratio : ", workload_ratio)
						}
						if (rows_spec[i][0] == "alg37p3"){
							workload_instance_ratio, _  = strconv.ParseFloat(rows_spec[i][1], 64) // rows_spec[i][1]:R's I ratio, [i][2]:B's I ratio, [i][3]:G's I, [i][4]:Y's I, [i][5]:S's I
							fmt.Print(" wja300 workload_instance_ratio : ", workload_instance_ratio)
						}
						if (rows_spec[i][0] == "amplification"){
							amplification, _ = strconv.ParseFloat(rows_spec[i][1], 64) 
							fmt.Print(" wja300 amplification : ", amplification)
						}
					}
					num_reqs = num_reqs*amplification
					num_reqs = num_reqs*workload_ratio/100*workload_instance_ratio/100 // alg3.7 i1 - resnet50 converting
					conv_reqs = float64(num_reqs) / reqs_base // i1 - resnet50
					if(request > int32(math.Ceil(conv_reqs))){
						// nothing done
					} else {
						request = int32(math.Ceil(conv_reqs))
					}
					fmt.Print(" wja300 num_reqs : ", num_reqs)
					fmt.Print(" wja300 conv_reqs : ", conv_reqs)
					fmt.Print(" wja300 request : ", request)
				}
		} else if (sstrings.Contains(apiName, "sentiment") && sstrings.Contains(apiName, "p3") && sstrings.Contains(apiName, "37") ){  // alg3.7 p3 - bert
				if ((period_second - time_correction) % 60 <= 9) { // every 1 minutes
					num_reqs, _ = strconv.ParseFloat(rows[row_count+4][1], 64)
					// weight conversion (alg3.7) num_reqs -> converting
					for i, _ := range rows_spec {
						if (rows_spec[i][0] == "p3B"){
							reqs_base, _ = strconv.ParseFloat(rows_spec[i][1], 64) // rows_sepc[i][1] = i1R's reqs_base
							fmt.Print(" wja300 B reqs_base : ", reqs_base)
						}
						if (rows_spec[i][0] == "ratio"){
							workload_ratio, _  = strconv.ParseFloat(rows_spec[i][2], 64) // rows_spec[i][1]:R'ratio, [i][2]:B, [i][3]:G, [i][4]:Y, [i][5]:S
							fmt.Print(" wja300 workload_ratio : ", workload_ratio)
						}
						if (rows_spec[i][0] == "alg37p3"){
							workload_instance_ratio, _  = strconv.ParseFloat(rows_spec[i][2], 64) // rows_spec[i][1]:R's I ratio, [i][2]:B's I ratio, [i][3]:G's I, [i][4]:Y's I, [i][5]:S's I
							fmt.Print(" wja300 workload_instance_ratio : ", workload_instance_ratio)
						}
						if (rows_spec[i][0] == "amplification"){
							amplification, _ = strconv.ParseFloat(rows_spec[i][1], 64) 
							fmt.Print(" wja300 amplification : ", amplification)
						}
					}
					num_reqs = num_reqs*amplification
					num_reqs = num_reqs*workload_ratio/100*workload_instance_ratio/100 // alg3.7 i1 - resnet50 converting
					conv_reqs = float64(num_reqs) / reqs_base // i1 - resnet50
					if(request > int32(math.Ceil(conv_reqs))){
						// nothing done
					} else {
						request = int32(math.Ceil(conv_reqs))
					}
					fmt.Print(" wja300 num_reqs : ", num_reqs)
					fmt.Print(" wja300 conv_reqs : ", conv_reqs)
					fmt.Print(" wja300 request : ", request)
				}
		} else if (sstrings.Contains(apiName, "text") && sstrings.Contains(apiName, "p3") && sstrings.Contains(apiName, "37") ){  // alg3.7 p3 - gpt
				if ((period_second - time_correction) % 60 <= 9) { // every 1 minutes
					num_reqs, _ = strconv.ParseFloat(rows[row_count+4][1], 64)
					// weight conversion (alg3.7) num_reqs -> converting
					for i, _ := range rows_spec {
						if (rows_spec[i][0] == "p3G"){
							reqs_base, _ = strconv.ParseFloat(rows_spec[i][1], 64) // rows_sepc[i][1] = i1R's reqs_base
							fmt.Print(" wja300 G reqs_base : ", reqs_base)
						}
						if (rows_spec[i][0] == "ratio"){
							workload_ratio, _  = strconv.ParseFloat(rows_spec[i][3], 64) // rows_spec[i][1]:R'ratio, [i][2]:B, [i][3]:G, [i][4]:Y, [i][5]:S
							fmt.Print(" wja300 workload_ratio : ", workload_ratio)
						}
						if (rows_spec[i][0] == "alg37p3"){
							workload_instance_ratio, _  = strconv.ParseFloat(rows_spec[i][3], 64) // rows_spec[i][1]:R's I ratio, [i][2]:B's I ratio, [i][3]:G's I, [i][4]:Y's I, [i][5]:S's I
							fmt.Print(" wja300 workload_instance_ratio : ", workload_instance_ratio)
						}
						if (rows_spec[i][0] == "amplification"){
							amplification, _ = strconv.ParseFloat(rows_spec[i][1], 64) 
							fmt.Print(" wja300 amplification : ", amplification)
						}
					}
					num_reqs = num_reqs*amplification
					num_reqs = num_reqs*workload_ratio/100*workload_instance_ratio/100 // alg3.7 i1 - resnet50 converting
					conv_reqs = float64(num_reqs) / reqs_base // i1 - resnet50
					if(request > int32(math.Ceil(conv_reqs))){
						// nothing done
					} else {
						request = int32(math.Ceil(conv_reqs))
					}
					fmt.Print(" wja300 num_reqs : ", num_reqs)
					fmt.Print(" wja300 conv_reqs : ", conv_reqs)
					fmt.Print(" wja300 request : ", request)
				}
		} else if (sstrings.Contains(apiName, "sound") && sstrings.Contains(apiName, "p3") && sstrings.Contains(apiName, "37") ){  // alg3.7 p3 - yamnet
				if ((period_second - time_correction) % 60 <= 9) { // every 1 minutes
					num_reqs, _ = strconv.ParseFloat(rows[row_count+4][1], 64)
					// weight conversion (alg3.7) num_reqs -> converting
					for i, _ := range rows_spec {
						if (rows_spec[i][0] == "p3Y"){
							reqs_base, _ = strconv.ParseFloat(rows_spec[i][1], 64) // rows_sepc[i][1] = i1R's reqs_base
							fmt.Print(" wja300 Y reqs_base : ", reqs_base)
						}
						if (rows_spec[i][0] == "ratio"){
							workload_ratio, _  = strconv.ParseFloat(rows_spec[i][4], 64) // rows_spec[i][1]:R'ratio, [i][2]:B, [i][3]:G, [i][4]:Y, [i][5]:S
							fmt.Print(" wja300 workload_ratio : ", workload_ratio)
						}
						if (rows_spec[i][0] == "alg37p3"){
							workload_instance_ratio, _  = strconv.ParseFloat(rows_spec[i][4], 64) // rows_spec[i][1]:R's I ratio, [i][2]:B's I ratio, [i][3]:G's I, [i][4]:Y's I, [i][5]:S's I
							fmt.Print(" wja300 workload_instance_ratio : ", workload_instance_ratio)
						}
						if (rows_spec[i][0] == "amplification"){
							amplification, _ = strconv.ParseFloat(rows_spec[i][1], 64) 
							fmt.Print(" wja300 amplification : ", amplification)
						}
					}
					num_reqs = num_reqs*amplification
					num_reqs = num_reqs*workload_ratio/100*workload_instance_ratio/100 // alg3.7 i1 - resnet50 converting
					conv_reqs = float64(num_reqs) / reqs_base // i1 - resnet50
					if(request > int32(math.Ceil(conv_reqs))){
						// nothing done
					} else {
						request = int32(math.Ceil(conv_reqs))
					}
					fmt.Print(" wja300 num_reqs : ", num_reqs)
					fmt.Print(" wja300 conv_reqs : ", conv_reqs)
					fmt.Print(" wja300 request : ", request)
				}
		} else if (sstrings.Contains(apiName, "inception") && sstrings.Contains(apiName, "p3") && sstrings.Contains(apiName, "37") ){  // alg3.7 p3 - inception
				if ((period_second - time_correction) % 60 <= 9) { // every 1 minutes
					num_reqs, _ = strconv.ParseFloat(rows[row_count+4][1], 64)
					// weight conversion (alg3.7) num_reqs -> converting
					for i, _ := range rows_spec {
						if (rows_spec[i][0] == "p3S"){
							reqs_base, _ = strconv.ParseFloat(rows_spec[i][1], 64) // rows_sepc[i][1] = i1R's reqs_base
							fmt.Print(" wja300 S reqs_base : ", reqs_base)
						}
						if (rows_spec[i][0] == "ratio"){
							workload_ratio, _  = strconv.ParseFloat(rows_spec[i][5], 64) // rows_spec[i][1]:R'ratio, [i][2]:B, [i][3]:G, [i][4]:Y, [i][5]:S
							fmt.Print(" wja300 workload_ratio : ", workload_ratio)
						}
						if (rows_spec[i][0] == "alg37p3"){
							workload_instance_ratio, _  = strconv.ParseFloat(rows_spec[i][5], 64) // rows_spec[i][1]:R's I ratio, [i][2]:B's I ratio, [i][3]:G's I, [i][4]:Y's I, [i][5]:S's I
							fmt.Print(" wja300 workload_instance_ratio : ", workload_instance_ratio)
						}
						if (rows_spec[i][0] == "amplification"){
							amplification, _ = strconv.ParseFloat(rows_spec[i][1], 64) 
							fmt.Print(" wja300 amplification : ", amplification)
						}
					}
					num_reqs = num_reqs*amplification
					num_reqs = num_reqs*workload_ratio/100*workload_instance_ratio/100 // alg3.7 i1 - resnet50 converting
					conv_reqs = float64(num_reqs) / reqs_base // i1 - resnet50
					if(request > int32(math.Ceil(conv_reqs))){
						// nothing done
					} else {
						request = int32(math.Ceil(conv_reqs))
					}
					fmt.Print(" wja300 num_reqs : ", num_reqs)
					fmt.Print(" wja300 conv_reqs : ", conv_reqs)
					fmt.Print(" wja300 request : ", request)
				}
		} else if (sstrings.Contains(apiName, "resnet50") && sstrings.Contains(apiName, "c5") && sstrings.Contains(apiName, "37") ){  // alg3.7 c5 - resnet50
				if ((period_second - time_correction) % 60 <= 9) { // every 1 minutes
					num_reqs, _ = strconv.ParseFloat(rows[row_count+4][1], 64)
					// weight conversion (alg3.7) num_reqs -> converting
					for i, _ := range rows_spec {
						if (rows_spec[i][0] == "c5R"){
							reqs_base, _ = strconv.ParseFloat(rows_spec[i][1], 64) // rows_sepc[i][1] = i1R's reqs_base
							fmt.Print(" wja300 R reqs_base : ", reqs_base)
						}
						if (rows_spec[i][0] == "ratio"){
							workload_ratio, _  = strconv.ParseFloat(rows_spec[i][1], 64) // rows_spec[i][1]:R'ratio, [i][2]:B, [i][3]:G, [i][4]:Y, [i][5]:S
							fmt.Print(" wja300 workload_ratio : ", workload_ratio)
						}
						if (rows_spec[i][0] == "alg37c5"){
							workload_instance_ratio, _  = strconv.ParseFloat(rows_spec[i][1], 64) // rows_spec[i][1]:R's I ratio, [i][2]:B's I ratio, [i][3]:G's I, [i][4]:Y's I, [i][5]:S's I
							fmt.Print(" wja300 workload_instance_ratio : ", workload_instance_ratio)
						}
						if (rows_spec[i][0] == "amplification"){
							amplification, _ = strconv.ParseFloat(rows_spec[i][1], 64) 
							fmt.Print(" wja300 amplification : ", amplification)
						}
					}
					num_reqs = num_reqs*amplification
					num_reqs = num_reqs*workload_ratio/100*workload_instance_ratio/100 // alg3.7 i1 - resnet50 converting
					conv_reqs = float64(num_reqs) / reqs_base // i1 - resnet50
					if(request > int32(math.Ceil(conv_reqs))){
						// nothing done
					} else {
						request = int32(math.Ceil(conv_reqs))
					}
					fmt.Print(" wja300 num_reqs : ", num_reqs)
					fmt.Print(" wja300 conv_reqs : ", conv_reqs)
					fmt.Print(" wja300 request : ", request)
				}
		} else if (sstrings.Contains(apiName, "sentiment") && sstrings.Contains(apiName, "c5") && sstrings.Contains(apiName, "37") ){  // alg3.7 c5 - bert
				if ((period_second - time_correction) % 60 <= 9) { // every 1 minutes
					num_reqs, _ = strconv.ParseFloat(rows[row_count+4][1], 64)
					// weight conversion (alg3.7) num_reqs -> converting
					for i, _ := range rows_spec {
						if (rows_spec[i][0] == "c5B"){
							reqs_base, _ = strconv.ParseFloat(rows_spec[i][1], 64) // rows_sepc[i][1] = i1R's reqs_base
							fmt.Print(" wja300 B reqs_base : ", reqs_base)
						}
						if (rows_spec[i][0] == "ratio"){
							workload_ratio, _  = strconv.ParseFloat(rows_spec[i][2], 64) // rows_spec[i][1]:R'ratio, [i][2]:B, [i][3]:G, [i][4]:Y, [i][5]:S
							fmt.Print(" wja300 workload_ratio : ", workload_ratio)
						}
						if (rows_spec[i][0] == "alg37c5"){
							workload_instance_ratio, _  = strconv.ParseFloat(rows_spec[i][2], 64) // rows_spec[i][1]:R's I ratio, [i][2]:B's I ratio, [i][3]:G's I, [i][4]:Y's I, [i][5]:S's I
							fmt.Print(" wja300 workload_instance_ratio : ", workload_instance_ratio)
						}
						if (rows_spec[i][0] == "amplification"){
							amplification, _ = strconv.ParseFloat(rows_spec[i][1], 64) 
							fmt.Print(" wja300 amplification : ", amplification)
						}
					}
					num_reqs = num_reqs*amplification
					num_reqs = num_reqs*workload_ratio/100*workload_instance_ratio/100 // alg3.7 i1 - resnet50 converting
					conv_reqs = float64(num_reqs) / reqs_base // i1 - resnet50
					if(request > int32(math.Ceil(conv_reqs))){
						// nothing done
					} else {
						request = int32(math.Ceil(conv_reqs))
					}
					fmt.Print(" wja300 num_reqs : ", num_reqs)
					fmt.Print(" wja300 conv_reqs : ", conv_reqs)
					fmt.Print(" wja300 request : ", request)
				}
		} else if (sstrings.Contains(apiName, "text") && sstrings.Contains(apiName, "c5") && sstrings.Contains(apiName, "37") ){  // alg3.7 c5 - gpt
				if ((period_second - time_correction) % 60 <= 9) { // every 1 minutes
					num_reqs, _ = strconv.ParseFloat(rows[row_count+4][1], 64)
					// weight conversion (alg3.7) num_reqs -> converting
					for i, _ := range rows_spec {
						if (rows_spec[i][0] == "c5G"){
							reqs_base, _ = strconv.ParseFloat(rows_spec[i][1], 64) // rows_sepc[i][1] = i1R's reqs_base
							fmt.Print(" wja300 G reqs_base : ", reqs_base)
						}
						if (rows_spec[i][0] == "ratio"){
							workload_ratio, _  = strconv.ParseFloat(rows_spec[i][3], 64) // rows_spec[i][1]:R'ratio, [i][2]:B, [i][3]:G, [i][4]:Y, [i][5]:S
							fmt.Print(" wja300 workload_ratio : ", workload_ratio)
						}
						if (rows_spec[i][0] == "alg37c5"){
							workload_instance_ratio, _  = strconv.ParseFloat(rows_spec[i][3], 64) // rows_spec[i][1]:R's I ratio, [i][2]:B's I ratio, [i][3]:G's I, [i][4]:Y's I, [i][5]:S's I
							fmt.Print(" wja300 workload_instance_ratio : ", workload_instance_ratio)
						}
						if (rows_spec[i][0] == "amplification"){
							amplification, _ = strconv.ParseFloat(rows_spec[i][1], 64) 
							fmt.Print(" wja300 amplification : ", amplification)
						}
					}
					num_reqs = num_reqs*amplification
					num_reqs = num_reqs*workload_ratio/100*workload_instance_ratio/100 // alg3.7 i1 - resnet50 converting
					conv_reqs = float64(num_reqs) / reqs_base // i1 - resnet50
					if(request > int32(math.Ceil(conv_reqs))){
						// nothing done
					} else {
						request = int32(math.Ceil(conv_reqs))
					}
					fmt.Print(" wja300 num_reqs : ", num_reqs)
					fmt.Print(" wja300 conv_reqs : ", conv_reqs)
					fmt.Print(" wja300 request : ", request)
				}
		} else if (sstrings.Contains(apiName, "sound") && sstrings.Contains(apiName, "c5") && sstrings.Contains(apiName, "37") ){  // alg3.7 c5 - yamnet
				if ((period_second - time_correction) % 60 <= 9) { // every 1 minutes
					num_reqs, _ = strconv.ParseFloat(rows[row_count+4][1], 64)
					// weight conversion (alg3.7) num_reqs -> converting
					for i, _ := range rows_spec {
						if (rows_spec[i][0] == "c5Y"){
							reqs_base, _ = strconv.ParseFloat(rows_spec[i][1], 64) // rows_sepc[i][1] = i1R's reqs_base
							fmt.Print(" wja300 Y reqs_base : ", reqs_base)
						}
						if (rows_spec[i][0] == "ratio"){
							workload_ratio, _  = strconv.ParseFloat(rows_spec[i][4], 64) // rows_spec[i][1]:R'ratio, [i][2]:B, [i][3]:G, [i][4]:Y, [i][5]:S
							fmt.Print(" wja300 workload_ratio : ", workload_ratio)
						}
						if (rows_spec[i][0] == "alg37c5"){
							workload_instance_ratio, _  = strconv.ParseFloat(rows_spec[i][4], 64) // rows_spec[i][1]:R's I ratio, [i][2]:B's I ratio, [i][3]:G's I, [i][4]:Y's I, [i][5]:S's I
							fmt.Print(" wja300 workload_instance_ratio : ", workload_instance_ratio)
						}
						if (rows_spec[i][0] == "amplification"){
							amplification, _ = strconv.ParseFloat(rows_spec[i][1], 64) 
							fmt.Print(" wja300 amplification : ", amplification)
						}
					}
					num_reqs = num_reqs*amplification
					num_reqs = num_reqs*workload_ratio/100*workload_instance_ratio/100 // alg3.7 i1 - resnet50 converting
					conv_reqs = float64(num_reqs) / reqs_base // i1 - resnet50
					if(request > int32(math.Ceil(conv_reqs))){
						// nothing done
					} else {
						request = int32(math.Ceil(conv_reqs))
					}
					fmt.Print(" wja300 num_reqs : ", num_reqs)
					fmt.Print(" wja300 conv_reqs : ", conv_reqs)
					fmt.Print(" wja300 request : ", request)
				}
		} else if (sstrings.Contains(apiName, "inception") && sstrings.Contains(apiName, "c5") && sstrings.Contains(apiName, "37") ){  // alg3.7 c5 - inception
				if ((period_second - time_correction) % 60 <= 9) { // every 1 minutes
					num_reqs, _ = strconv.ParseFloat(rows[row_count+4][1], 64)
					// weight conversion (alg3.7) num_reqs -> converting
					for i, _ := range rows_spec {
						if (rows_spec[i][0] == "c5S"){
							reqs_base, _ = strconv.ParseFloat(rows_spec[i][1], 64) // rows_sepc[i][1] = i1R's reqs_base
							fmt.Print(" wja300 S reqs_base : ", reqs_base)
						}
						if (rows_spec[i][0] == "ratio"){
							workload_ratio, _  = strconv.ParseFloat(rows_spec[i][5], 64) // rows_spec[i][1]:R'ratio, [i][2]:B, [i][3]:G, [i][4]:Y, [i][5]:S
							fmt.Print(" wja300 workload_ratio : ", workload_ratio)
						}
						if (rows_spec[i][0] == "alg37c5"){
							workload_instance_ratio, _  = strconv.ParseFloat(rows_spec[i][5], 64) // rows_spec[i][1]:R's I ratio, [i][2]:B's I ratio, [i][3]:G's I, [i][4]:Y's I, [i][5]:S's I
							fmt.Print(" wja300 workload_instance_ratio : ", workload_instance_ratio)
						}
						if (rows_spec[i][0] == "amplification"){
							amplification, _ = strconv.ParseFloat(rows_spec[i][1], 64) 
							fmt.Print(" wja300 amplification : ", amplification)
						}
					}
					num_reqs = num_reqs*amplification
					num_reqs = num_reqs*workload_ratio/100*workload_instance_ratio/100 // alg3.7 i1 - resnet50 converting
					conv_reqs = float64(num_reqs) / reqs_base // i1 - resnet50
					if(request > int32(math.Ceil(conv_reqs))){
						// nothing done
					} else {
						request = int32(math.Ceil(conv_reqs))
					}
					fmt.Print(" wja300 num_reqs : ", num_reqs)
					fmt.Print(" wja300 conv_reqs : ", conv_reqs)
					fmt.Print(" wja300 request : ", request)
				}
		}



// autoscaling control /////////////////////////////////////////////////////////////////////////////////////////////////////

		if ((period_second - time_correction) % 60 <= 9){

			if currentReplicas != request {
				apiLogger.Infof("%s autoscaling event: %d -> %d", apiName, currentReplicas, request)

				deployment, err := config.K8s.GetDeployment(initialDeployment.Name)
				if err != nil {
					return err
				}

				if deployment == nil {
					return errors.ErrorUnexpected("unable to find k8s deployment", apiName)
				}

				deployment.Spec.Replicas = &request

				if _, err := config.K8s.UpdateDeployment(deployment); err != nil {
					return err
				}

				currentReplicas = request
			}
		}


		return nil
	}, nil

}
