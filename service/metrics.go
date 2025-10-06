package main

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"go.uber.org/zap"
)

// CloudWatchMetrics handles CloudWatch metrics operations
type CloudWatchMetrics struct {
	cw     *cloudwatch.CloudWatch
	logger *zap.Logger
}

// sendRouteMetrics sends route metrics to CloudWatch
func (m *CloudWatchMetrics) sendRouteMetrics(endpoint string, statusCode int, duration time.Duration) {
	namespace := "GoObservabilityDemo/Application"

	metrics := []*cloudwatch.MetricDatum{
		{
			MetricName: aws.String("RequestDuration"),
			Value:      aws.Float64(duration.Seconds()),
			Unit:       aws.String("Seconds"),
			Dimensions: []*cloudwatch.Dimension{
				{
					Name:  aws.String("Endpoint"),
					Value: aws.String(endpoint),
				},
			},
			Timestamp: aws.Time(time.Now()),
		},
		{
			MetricName: aws.String("RequestCount"),
			Value:      aws.Float64(1),
			Unit:       aws.String("Count"),
			Dimensions: []*cloudwatch.Dimension{},
			Timestamp:  aws.Time(time.Now()),
		},
		{
			MetricName: aws.String("RequestCount"),
			Value:      aws.Float64(1),
			Unit:       aws.String("Count"),
			Dimensions: []*cloudwatch.Dimension{
				{
					Name:  aws.String("Endpoint"),
					Value: aws.String(endpoint),
				},
			},
			Timestamp: aws.Time(time.Now()),
		},
	}

	_, err := m.cw.PutMetricData(&cloudwatch.PutMetricDataInput{
		Namespace:  aws.String(namespace),
		MetricData: metrics,
	})

	if err != nil {
		m.logger.Error("Failed to send CloudWatch metrics", zap.Error(err))
	}
}

// sendCreatedCoffeeOrderMetrics sends coffee order creation metrics to CloudWatch
func (m *CloudWatchMetrics) sendCreatedCoffeeOrderMetrics(coffeeType string) {
	namespace := "GoObservabilityDemo/Application"

	metrics := []*cloudwatch.MetricDatum{
		{
			MetricName: aws.String("CreatedCoffeeOrders_Total"),
			Value:      aws.Float64(1),
			Unit:       aws.String("Count"),
			Dimensions: []*cloudwatch.Dimension{},
			Timestamp:  aws.Time(time.Now()),
		},
		{
			MetricName: aws.String("CreatedCoffeeOrders_ByType"),
			Value:      aws.Float64(1),
			Unit:       aws.String("Count"),
			Dimensions: []*cloudwatch.Dimension{
				{
					Name:  aws.String("CoffeeType"),
					Value: aws.String(coffeeType),
				},
			},
			Timestamp: aws.Time(time.Now()),
		},
	}

	_, err := m.cw.PutMetricData(&cloudwatch.PutMetricDataInput{
		Namespace:  aws.String(namespace),
		MetricData: metrics,
	})

	if err != nil {
		m.logger.Error("Failed to send created coffee order metric to CloudWatch", zap.Error(err))
	}
}
