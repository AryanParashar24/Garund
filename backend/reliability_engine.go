package main

type ReliabilityMeasurement struct {
	Value       *float64
	GoodEvents  int64
	TotalEvents int64
	Status      string
}

func calculateAvailabilitySLI(
	successRate float64,
	totalRate float64,
	target float64,
	available bool,
) ReliabilityMeasurement {

	if !available || totalRate <= 0 {
		return ReliabilityMeasurement{
			Status: "unavailable",
		}
	}

	value := successRate / totalRate * 100

	return ReliabilityMeasurement{
		Value:       &value,
		GoodEvents:  int64(successRate),
		TotalEvents: int64(totalRate),
		Status: reliabilityStatus(
			value,
			target,
		),
	}
}

func calculateErrorRateSLI(
	errorRate float64,
	totalRate float64,
	target float64,
	available bool,
) ReliabilityMeasurement {

	if !available || totalRate <= 0 {
		return ReliabilityMeasurement{
			Status: "unavailable",
		}
	}

	value := errorRate / totalRate * 100

	status := "healthy"

	if value > target {
		status = "critical"
	}

	return ReliabilityMeasurement{
		Value: &value,
		GoodEvents: int64(
			totalRate - errorRate,
		),
		TotalEvents: int64(totalRate),
		Status:      status,
	}
}

func calculateLatencySLI(
	value float64,
	target float64,
	available bool,
) ReliabilityMeasurement {

	if !available {
		return ReliabilityMeasurement{
			Status: "unavailable",
		}
	}

	status := "healthy"

	if value > target {
		status = "critical"
	}

	return ReliabilityMeasurement{
		Value:  &value,
		Status: status,
	}
}
