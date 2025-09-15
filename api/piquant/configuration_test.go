package piquant

import "fmt"

func Example_piquant_ReadFieldFromPIQUANTConfigMSA() {
	piquantMSA := `##INCSR      : 0.0152   Solid angle from source in steradians (can include normalization for optic file - use this for tuning 0.00355)
##INCANGLE   : 90.00   Incident angle of primary X-ray beam in degrees (90 is normal incidence)
#ELEVANGLE   : 48.03   Elevation angle of detector, in degrees (90 is normal to surface)
#AZIMANGLE   : 180.0   Azimuth angle between incident beam plane and detected beam plane
##GEOMETRY   : 1.0     Geometric correction factor
#SOLIDANGLE  : 0.224 Solid angle collected by the detector in steradians`

	a, err := ReadFieldFromPIQUANTConfigMSA(piquantMSA, "ELEVANGLE")
	fmt.Printf("%v|%v", a, err)

	// Output:
	// 48.03|<nil>
}
