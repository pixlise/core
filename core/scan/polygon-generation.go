package scan

import (
	"errors"
	"fmt"
	"math"

	"github.com/engelsjk/polygol"
	vornoi "github.com/haddock7/voronoi"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func GeneratePolygons(imageName string,
	scanItem *protos.ScanItem,
	scanEntries []*protos.ScanEntry,
	beamXYZs []*protos.Coordinate3D,
	beamIJs *[]*protos.Coordinate2D,
	detectorConfig *protos.DetectorConfig,
) (*protos.ImageScanEntryDisplayElementsGetResp, error) {
	g := gen{}
	scanPoints, err := g.initLocationCachingForBeams(scanItem.Instrument, scanEntries, beamXYZs, beamIJs, imageName, true)
	if err != nil {
		return nil, err
	}

	if g.locationCount <= 0 {
		return nil, fmt.Errorf("Failed to generate scan points for scan: %v", scanItem.Id)
	}

	beamUnitsInMeters := decideBeamUnitsIsMeters(scanItem.Instrument, g.locationPointZMax)

	if len(imageName) <= 0 && beamIJs != nil {
		// If we don't have an image, we don't scale down
		beamUnitsInMeters = false
	}

	beamRadius_mm := float64(detectorConfig.MmBeamRadius)
	if beamRadius_mm == 0 {
		// If we don't get one, use the default
		beamRadius_mm = 0.06
	}

	contextPixelsTommConversion := float64(-1)
	if scanItem.Instrument != protos.ScanInstrument_UNKNOWN_INSTRUMENT {
		contextPixelsTommConversion = g.calcImagePixelsToPhysicalmm(beamUnitsInMeters)
	}

	fmt.Printf("  Conversion factor for image pixels to mm: %v\n", contextPixelsTommConversion)

	//beamRadius_pixels := beamRadius_mm / contextPixelsTommConversion
	g.findMinPointDistances(scanPoints, scanEntries, beamXYZs, beamUnitsInMeters)

	clusters, err := g.makePointClusters(scanPoints, scanItem.Instrument == protos.ScanInstrument_UNKNOWN_INSTRUMENT)
	if err != nil {
		return nil, err
	}

	// Clear footprints, get from clusters as we process them
	wholeFootprintHullPoints := [][]HullPoint{}

	// Allocate blank polygons for each...
	scanPointPolygons := []ScanPointPolygon{}
	for c := 0; c < len(scanPoints); c++ {
		scanPointPolygons = append(scanPointPolygons, ScanPointPolygon{})
	}

	for _, cluster := range clusters {
		// NOTE: This 50 might be redundant but we had it historically here so left it in while working on
		//       the 3d version of this using 0, can remove the value if it has no effect
		makeScanPointPolygons(50, cluster, scanPoints, scanPointPolygons)
		wholeFootprintHullPoints = append(wholeFootprintHullPoints, cluster.FootprintPoints)
	}

	// Convert to proto-compatible structures... might be worth doing it this way initially, but this provided
	// a perfect 1:1 typescript->Go conversion up to this point.

	protoClusters := []*protos.PointCluster{}
	for _, cl := range clusters {
		idxs := make([]uint64, len(cl.LocIdxs))
		for c, i := range cl.LocIdxs {
			idxs[c] = uint64(i)
		}

		hullPts := makeProtoHullPoints(cl.FootprintPoints)
		protoClusters = append(protoClusters, &protos.PointCluster{
			ScanEntryIndexes:     idxs,
			AveragePointDistance: cl.PointDistance,
			AngleRadiansToImage:  cl.AngleRadiansToContextImage,
			FootprintPoints:      hullPts,
		})
	}

	protoPolys := []*protos.ScanEntryPolygon{}
	for _, p := range scanPointPolygons {
		pts := []*protos.Coordinate2D{}
		for _, pt := range p.Points {
			pts = append(pts, &protos.Coordinate2D{I: float32(pt.X), J: float32(pt.Y)})
		}

		protoPolys = append(protoPolys, &protos.ScanEntryPolygon{
			Points: pts,
			Bbox:   makeProtoRect(p.BBox),
		})
	}

	protoFootprints := []*protos.Footprint{}
	for _, f := range wholeFootprintHullPoints {
		hullPts := makeProtoHullPoints(f)
		protoFootprints = append(protoFootprints, &protos.Footprint{
			HullPoints: hullPts,
		})
	}

	protoScanPoints := []*protos.ScanPoint{}
	for _, pt := range scanPoints {
		sendPt := &protos.ScanPoint{
			PMC:                  uint64(pt.PMC),
			Coord:                nil,
			LocationIdx:          uint64(pt.locationIdx),
			HasNormalSpectra:     pt.hasNormalSpectra,
			HasDwellSpectra:      pt.hasDwellSpectra,
			HasPseudoIntensities: pt.hasPseudoIntensities,
			HasMissingData:       pt.hasMissingData,
		}
		if pt.coord != nil {
			sendPt.Coord = &protos.Coordinate2D{I: float32(pt.coord.X), J: float32(pt.coord.Y)}
		}
		protoScanPoints = append(protoScanPoints, sendPt)
	}

	resp := &protos.ImageScanEntryDisplayElementsGetResp{
		ScanPoints:             protoScanPoints,
		PointClusters:          protoClusters,
		ScanEntryPolygons:      protoPolys,
		Footprints:             protoFootprints,
		PixelToMMConversion:    contextPixelsTommConversion,
		ScanPointDisplayRadius: g.locationDisplayPointRadius,
		ScanPointBBox:          makeProtoRect(g.locationPointBBox),
		BeamRadiusMM:           beamRadius_mm,
	}

	return resp, nil
}

func makeProtoRect(r Rect) *protos.Rectangle {
	return &protos.Rectangle{
		X: r.X,
		Y: r.Y,
		W: r.W,
		H: r.H,
	}
}

func makeProtoHullPoints(pts []HullPoint) []*protos.HullPoint {
	hullPts := []*protos.HullPoint{}
	for _, p := range pts {
		hullPts = append(hullPts, &protos.HullPoint{
			Point:          &protos.Coordinate2D{I: float32(p.Point.X), J: float32(p.Point.Y)},
			Normal:         &protos.Coordinate2D{I: float32(p.Normal.X), J: float32(p.Normal.Y)},
			ScanEntryIndex: uint64(p.Idx),
		})
	}
	return hullPts
}

type PointCluster struct {
	LocIdxs                    []int
	PointDistance              float64
	FootprintPoints            []HullPoint
	AngleRadiansToContextImage float64
}

func isClusterScanPoint(pt ScanPoint) bool {
	if pt.coord == nil || (!pt.hasNormalSpectra && !pt.hasPseudoIntensities) {
		// No coord, won't have spectra either... ignore
		return false
	}
	return true
}

// Finds points that are clustered nearby and returns their location indexes
// Currently the only place this really happens is the cal target scans where we
// take several lines and grids with large jumps between them.
// Because PIXL goes sequentially through PMCs, we just need to find when there
// is a large gap between scan points
func (g *gen) makePointClusters(scanPoints []ScanPoint, treateAsSingleCluster bool) ([]PointCluster, error) {
	// Loop through locations, if distance jump is significantly larger than last size, we
	// assume a new cluster of points has started
	clusters := []PointCluster{PointCluster{}}

	if treateAsSingleCluster {
		// Create a single cluster and find an average point distance to use
		var ptDistance float64
		ptDistCount := 0

		lastIdx := -1
		for locIdx := 0; locIdx < len(scanPoints); locIdx++ {
			if isClusterScanPoint(scanPoints[locIdx]) {
				clusters[0].LocIdxs = append(clusters[0].LocIdxs, locIdx)

				if lastIdx > -1 && ptDistCount < 20 {
					vec := subtractVectors(*scanPoints[lastIdx].coord, *scanPoints[locIdx].coord)
					dst := getVectorLength(vec)

					ptDistance += dst
					ptDistCount++
				}

				lastIdx = locIdx
			}
		}

		clusters[0].PointDistance = 1
		if ptDistCount > 0 {
			clusters[0].PointDistance = ptDistance / float64(ptDistCount)
		}
	} else {
		clusters = breakIntoClustersPIXLStyle(scanPoints)
	}

	// If we only have the 1 default cluster we added...
	if len(clusters) == 1 && len(clusters[0].LocIdxs) <= 0 {
		clusters = []PointCluster{}
	}

	// Calculate footprints for all clusters
	for c := 0; c < len(clusters); c++ {
		cluster := &clusters[c]
		cluster.FootprintPoints = g.makeConvexHull(cluster.LocIdxs, scanPoints)
		if angle, err := findExperimentAngle(cluster.FootprintPoints); err != nil {
			return clusters, err
		} else {
			cluster.AngleRadiansToContextImage = angle
		}

		cluster.FootprintPoints = fattenFootprint(
			cluster.FootprintPoints,
			cluster.PointDistance/2,
			cluster.AngleRadiansToContextImage,
		)

		fmt.Printf(
			`  Point cluster %v contains %v PMCs, %v footprint points, %.3f degrees rotated
`, c+1, len(cluster.LocIdxs), len(cluster.FootprintPoints), cluster.AngleRadiansToContextImage*180/math.Pi)
	}

	return clusters, nil
}

func fattenFootprint(footprintHullPoints []HullPoint, enlargeBy float64, angleRad float64) []HullPoint {
	if len(footprintHullPoints) <= 0 {
		fmt.Printf("  WARNING: Footprint hull not widened, no points exist")
		return []HullPoint{}
	}

	// If it's a line scan, we may have ended up with a hull that's basically 2 parallel lines (or close to it).
	// We want this to be expanded out, so at this point we take all hull points and find the hull of all of those points if
	// they were formed of a rect

	// Make rotated boxes for each point, then form the hull around it
	centers := []Point{}
	for _, pt := range footprintHullPoints {
		centers = append(centers, Point{pt.Point.X, pt.Point.Y})
	}

	boxes := makeRotatedBoxes(centers, enlargeBy, angleRad)

	fatHullPoints := []HullPoint{}
	for c, box := range boxes {
		for _, pt := range box {
			fatHullPoints = append(fatHullPoints, HullPoint{Point: pt, Idx: footprintHullPoints[c].Idx})
		}
	}

	result, err := QuickHull(fatHullPoints)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	}

	calcFootprintNormals(result)
	return result
}

func calcFootprintNormals(footprintHullPoints []HullPoint) {
	if len(footprintHullPoints) <= 0 {
		fmt.Println("  Footprint hull normals not calculated, no points exist")
		return
	}

	// Calc normals so we can draw expanded
	normals := []Point{}
	for c := 0; c < len(footprintHullPoints); c++ {
		nextPtIdx := c + 1
		if c == len(footprintHullPoints)-1 {
			nextPtIdx = 0
		}

		nextPt := footprintHullPoints[nextPtIdx]

		lineVec := normalizeVector(getVectorBetweenPoints(footprintHullPoints[c].Point, nextPt.Point))
		normals = append(normals, Point{lineVec.Y, -lineVec.X})
	}

	// Smooth them and save
	for c := 0; c < len(footprintHullPoints); c++ {
		lastIdx := c - 1
		if lastIdx < 0 {
			lastIdx = len(footprintHullPoints) - 1
		}

		N := normalizeVector(addVectors(normals[c], normals[lastIdx]))
		footprintHullPoints[c].Normal = &Point{}
		*footprintHullPoints[c].Normal = N
	}
}

func makeRotatedBoxes(centers []Point, halfSideLength float64, angleRad float64) [][]Point {
	// Calculate vectors to add to each center to form the box
	xAddVec := Point{halfSideLength, 0}
	yAddVec := Point{0, halfSideLength}

	// Rotate them by the experiment angle
	rotM := getRotationMatrix(angleRad)

	/* angle -0.6960058151466897
		         -0.6960058151466897

			matrix:
			[0.767409204019723, 0.64115763552017, 0]
		    [-0.64115763552017, 0.767409204019723, 0]
		    [0, 0, 1]

			[0.767409204019723,0.64115763552017,0]
	        [-0.64115763552017,0.767409204019723,0]
	        [0,0,1]

			pt: 0.25164135480895305, 0

				x sum = (matrixrow[0][0]=0.767 * bdata[0][0]=0.25) = 0.19 + (matrixrow[0][1]=0.64 * bdata[1][0]=0) = 0 + (matrixrow[0][2]=0 * bdata[2][0]=1) = 0 = 0.19
				y sum = (matrixrow[1][0]=-0.64 * bdata[0][0]=0.25) = -0.16 + (matrixrow[1][1]=0.767 * bdata[1][0]=0) = 0 + (matrixrow[1][2]=0 * bdata[2][0]=1) = 0 = -0.16
	*/

	xAddRotatedVec := pointByMatrix(rotM, xAddVec)
	yAddRotatedVec := pointByMatrix(rotM, yAddVec)

	// Calc the negative direction too
	xSubRotatedVec := subtractVectors(Point{}, xAddRotatedVec)
	ySubRotatedVec := subtractVectors(Point{}, yAddRotatedVec)

	boxes := [][]Point{}

	for _, center := range centers {
		// Calculate the 4 corners of the box around this center
		box := []Point{
			addVectors(addVectors(center, xAddRotatedVec), yAddRotatedVec),
			addVectors(addVectors(center, xSubRotatedVec), yAddRotatedVec),
			addVectors(addVectors(center, xSubRotatedVec), ySubRotatedVec),
			addVectors(addVectors(center, xAddRotatedVec), ySubRotatedVec),
		}

		boxes = append(boxes, box)
	}

	return boxes
}

// Returns the experiment angle in radians.
// Can be called any time
func findExperimentAngle(footprintHullPoints []HullPoint) (float64, error) {
	var experimentAngleRad float64

	if len(footprintHullPoints) <= 0 {
		fmt.Println("  Experiment angle not checked, as no location data exists")
		return experimentAngleRad, nil
	}

	// Now that we have a hull, we can find the experiment angle. To do this we take the longest edge of the
	// hull and use the angle formed by that vs the X axis
	var longestVec *Point
	var longestVecLength float64

	for c := 0; c < len(footprintHullPoints); c++ {
		lastIdx := c - 1
		if lastIdx < 0 {
			lastIdx = len(footprintHullPoints) - 1
		}

		vec := getVectorBetweenPoints(footprintHullPoints[lastIdx].Point, footprintHullPoints[c].Point)
		vecLen := getVectorLength(vec)
		if longestVec == nil || vecLen > longestVecLength {
			longestVec = &vec
			longestVecLength = vecLen
		}
	}

	if longestVec == nil {
		// Just return 0
		return 0, fmt.Errorf("  findExperimentAngle failed, so using 0 degrees")
	}

	// Now find how many degrees its rotated relative to X axis
	normalVec := normalizeVector(*longestVec)

	// Calculate angle
	experimentAngleRad = math.Acos(float64(getVectorDotProduct(Point{0, -1}, normalVec)))

	if normalVec.X < 0 {
		experimentAngleRad = math.Pi/2 - experimentAngleRad
	}

	// If the angle is near 90, 180, 270 or 360, set it to 0 so we don't
	// pointlessly do the rotation when drawing rectangles!
	angleDeg := experimentAngleRad * 180 / math.Pi

	// If it's near 90 increments, set to 0
	if math.Abs(angleDeg) < 5 || math.Abs(angleDeg-90) < 5 || math.Abs(angleDeg-270) < 5 || math.Abs(angleDeg-360) < 5 {
		angleDeg = 0
		experimentAngleRad = 0
	}

	return experimentAngleRad, nil
}

func (g *gen) makeConvexHull(useLocIdxs []int, scanPoints []ScanPoint) []HullPoint {
	hullPoints := []HullPoint{}
	for _, locIdx := range useLocIdxs {
		loc := scanPoints[locIdx]

		if loc.coord != nil && (loc.hasNormalSpectra || loc.hasPseudoIntensities) {
			// normal spectra may not be down yet!
			hullPoints = append(hullPoints, HullPoint{Point: *loc.coord, Idx: locIdx})
		}
	}

	hull, err := QuickHull(hullPoints)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	}

	return hull
}

func breakIntoClustersPIXLStyle(scanPoints []ScanPoint) []PointCluster {
	// Loop through locations, if distance jump is significantly larger than last size, we
	// assume a new cluster of points has started
	clusters := []PointCluster{PointCluster{}}

	lastIdx := -1
	lastDistance := float64(-1)
	var distanceSum float64
	nonZeroDistanceCount := 0

	// We keep track of the angle at which the gap that broke the cluster went. This is so we
	// can detect the case where for eg breadboards are scanning in a Z shape, so every line
	// moves to the start of the previous line, hence there is a large (same angled) leap.
	// If this is the case, the special work-around is to just return the whole thing as one cluster.
	clusterBreakAngleCosines := []float64{}

	for locIdx := 0; locIdx < len(scanPoints); locIdx++ {
		if !isClusterScanPoint(scanPoints[locIdx]) {
			// No coord, won't have spectra either... ignore
			continue
		}

		// If we've seen one already, do a distance compare
		if lastIdx >= 0 {
			vec := subtractVectors(*scanPoints[lastIdx].coord, *scanPoints[locIdx].coord)
			dst := getVectorLength(vec)
			if lastDistance > -1 && dst > (distanceSum/float64(nonZeroDistanceCount))*10 {
				// Save the point distance for the last cluster
				lastCluster := &clusters[len(clusters)-1]

				lastCluster.PointDistance = distanceSum
				if nonZeroDistanceCount > 0 {
					lastCluster.PointDistance /= float64(nonZeroDistanceCount)
				}

				// Save the angle at which this break happened
				clusterBreakAngleCosines = append(clusterBreakAngleCosines, getVectorDotProduct(normalizeVector(vec), Point{1, 0}))

				// Start a new cluster!
				clusters = append(clusters, PointCluster{})

				// Forget last distance, we need to discover a new one now
				lastDistance = -1
				distanceSum = 0
				nonZeroDistanceCount = 0
			} else {
				if dst > 0 {
					lastDistance = dst

					distanceSum += dst
					nonZeroDistanceCount++
				}
			}
		}

		clusters[len(clusters)-1].LocIdxs = append(clusters[len(clusters)-1].LocIdxs, locIdx)
		lastIdx = locIdx
	}

	// Calculate distance for the last one
	if len(clusters) > 0 {
		lastCluster := &clusters[len(clusters)-1]

		lastCluster.PointDistance = distanceSum
		if nonZeroDistanceCount > 0 {
			lastCluster.PointDistance /= float64(nonZeroDistanceCount)
		}
	}

	// If we find that the clusters are all broken in the same direction, we have to assume it's a scan done in a Z pattern, and we
	// don't want every single scan line to be a separate cluster, so here we check for that and if that's the case, we build one single
	// cluster for the whole thing
	if len(clusterBreakAngleCosines) > 0 {
		similarAngleCount := 0
		for _, angleCos := range clusterBreakAngleCosines {
			// We allow for some floating-point accuracy mess, but really they should be exactly equal
			if math.Abs(float64(angleCos-clusterBreakAngleCosines[0])) < 0.001 {
				similarAngleCount++
			}
		}

		if similarAngleCount >= len(clusterBreakAngleCosines) {
			// We are assuming this is a Z scan pattern, so we turn the whole thing into a single cluster
			singleCluster := PointCluster{[]int{}, clusters[0].PointDistance, []HullPoint{}, clusters[0].AngleRadiansToContextImage}
			for _, cluster := range clusters {
				singleCluster.LocIdxs = append(singleCluster.LocIdxs, cluster.LocIdxs...)
			}

			clusters = []PointCluster{singleCluster}
		}
	}

	return clusters
}

// Sets some local stats about point coordinates:
// locationDisplayPointRadius, minXYDistance_mm
func (g *gen) findMinPointDistances(scanPoints []ScanPoint, scanEntries []*protos.ScanEntry, beamLocations []*protos.Coordinate3D, beamUnitsInMeters bool) error {
	if g.locationCount <= 0 {
		return errors.New("findMinPointDistances with no location data")
	}

	//NumSamples := 1000

	// Randomly pick a few points, find the min distance to between any other point to that point
	// and then average this out
	samples := []int{}
	nearestDistanceToSamples := []float64{}
	/*
		for c := 0; c < NumSamples; c++ {
			sampleIdx := int64(-1)

			// Make sure it's got a location
			for sampleIdx < 0 {
				sampleIdx = rand.Int63() % int64(len(scanPoints)-1)
				if !scanEntries[sampleIdx].Location {
					sampleIdx = -1
				}
			}

			samples = append(samples, int(sampleIdx))
		}
	*/

	for c := 0; c < len(scanEntries); c++ {
		if scanEntries[c].Location {
			samples = append(samples, c)
		}
	}

	// Now loop through all and find the nearest point to each sample in distance-squared units
	ExclusionBoxSize := float64(g.locationPointBBox.W+g.locationPointBBox.H) / 2 / 10

	for _, sampleIdx := range samples {
		samplePt := scanPoints[sampleIdx].coord

		nearestIdx := -1
		nearestDistSq := ExclusionBoxSize * ExclusionBoxSize

		// Find the distance of the nearest point - we can exclude most of the points fast by bounding box
		locIdx := 0
		for _, locPt := range scanPoints {
			// Don't compare to itself, don't compare to PMCs without locations!
			if locPt.coord != nil && locIdx != sampleIdx {
				xDiff := math.Abs(float64(samplePt.X - locPt.coord.X))
				yDiff := math.Abs(float64(samplePt.Y - locPt.coord.Y))

				// Could use ptWithinBox but then gotta calculate xDiff and yDiff anyway...

				if xDiff < ExclusionBoxSize && yDiff < ExclusionBoxSize {
					// Get the square distance
					distSq := xDiff*xDiff + yDiff*yDiff
					if distSq < nearestDistSq {
						nearestIdx = locIdx
						nearestDistSq = distSq
					}
				}
			}
			locIdx++
		}

		if nearestIdx >= 0 {
			nearestDistanceToSamples = append(nearestDistanceToSamples, math.Sqrt(float64(nearestDistSq)))
		}
	}

	// Now we have an array of nearest distances, average them and get to a single radius to use
	var totalDist float64
	for _, dist := range nearestDistanceToSamples {
		totalDist += dist
	}
	g.locationDisplayPointRadius = totalDist / float64(len(nearestDistanceToSamples))

	// Increase it a bit, to make sure things are covered nicely
	g.locationDisplayPointRadius *= 1.1

	if math.IsNaN(float64(g.locationDisplayPointRadius)) {
		g.locationDisplayPointRadius = 1
	}

	fmt.Printf("  Generated locationDisplayPointRadius: %v\n", g.locationDisplayPointRadius)

	// The above was done in image space (context image pixels, i/j coordinates). We now do the same in physical XYZ coordinates
	g.minXYDistance_mm = g.locationPointXSize + g.locationPointYSize + g.locationPointZSize

	for c, scanEntry := range scanEntries {
		// We're only interested if there are spectra (or pseudo-intensities, as we may not have received the spectra yet)
		if scanEntry.Location && (scanEntry.NormalSpectra > 0 || scanEntry.PseudoIntensities) {
			cPt := Point{float64(beamLocations[c].X), float64(beamLocations[c].Y)}

			for i := c + 1; i < len(scanEntries); i++ {
				// We're only interested if there are spectra!
				if scanEntries[i].Location && (scanEntries[i].NormalSpectra > 0 || scanEntries[i].PseudoIntensities) {
					iPt := Point{float64(beamLocations[i].X), float64(beamLocations[i].Y)}

					vec := getVectorBetweenPoints(cPt, iPt)

					distSq := getVectorDotProduct(vec, vec)
					if distSq > 0 && distSq < g.minXYDistance_mm {
						g.minXYDistance_mm = distSq
					}
				}
			}
		}

		g.minXYDistance_mm = math.Sqrt(float64(g.minXYDistance_mm))

		// If we're in meters, convert
		// TODO: This is potentially wrong, looking at the code well after it was written we could just do
		// this conversion after the for loop but it might affect the result!
		if beamUnitsInMeters {
			g.minXYDistance_mm *= 1000
		}
	}

	return nil
}

// Returns the conversion multiplier to go from context image pixels to physical units in mm (based on beam location)
func (g *gen) calcImagePixelsToPhysicalmm(beamUnitsInMeters bool) float64 {
	// We see the diagonal size of the location points bbox vs the widest X distance between points
	mmConversion := math.Sqrt(
		float64(g.locationPointXSize*g.locationPointXSize+g.locationPointYSize*g.locationPointYSize) /
			float64(g.locationPointBBox.W*g.locationPointBBox.W+g.locationPointBBox.H*g.locationPointBBox.H))

	if beamUnitsInMeters {
		mmConversion *= 1000.0
	}

	return mmConversion
}

func decideBeamUnitsIsMeters(scanInstrument protos.ScanInstrument, locPointZMaxValue float64) bool {
	// Units in the beam location file were converted from mm to meters around June 2020, the way to tell what
	// we're dealing with is by Z, as our standoff distance is always around 25mm, so in mm units this is > 1
	// and in m it's way < 1
	beamInMeters := (scanInstrument == protos.ScanInstrument_PIXL_FM || scanInstrument == protos.ScanInstrument_PIXL_EM) && locPointZMaxValue < 1.0
	if beamInMeters {
		fmt.Println("  Beam location is in meters")
	} else {
		fmt.Println("  Beam location is in mm")
	}
	return beamInMeters
}

type ScanPoint struct {
	PMC                  uint32
	coord                *Point
	locationIdx          int32
	hasNormalSpectra     bool
	hasDwellSpectra      bool
	hasPseudoIntensities bool
	hasMissingData       bool
}

type gen struct {
	locationCount              int32
	locationsWithNormalSpectra int32

	locationPointXSize float64
	locationPointYSize float64
	locationPointZSize float64

	locationPointZMax float64

	locationPointBBox Rect

	locationDisplayPointRadius float64
	minXYDistance_mm           float64
}

func (g *gen) initLocationCachingForBeams(
	detector protos.ScanInstrument,
	scanEntries []*protos.ScanEntry,
	beamLocations []*protos.Coordinate3D,
	beamIJs *[]*protos.Coordinate2D,
	imageName string,
	readBeamIJSwapped bool,
) ([]ScanPoint, error) {
	scanPoints := []ScanPoint{}
	g.locationCount = 0
	g.locationsWithNormalSpectra = 0

	locPointXMinMax := MinMax{}
	locPointYMinMax := MinMax{}
	locPointZMinMax := MinMax{}

	// At this point, check that the arrays we have for all scan data have the same sizes
	// because the theory is that we can index across them
	if len(scanEntries) != len(beamLocations) {
		return []ScanPoint{}, fmt.Errorf(`ScanEntry length %v doesn't match beam location length %v`, len(scanEntries), len(beamLocations))
	}
	if beamIJs != nil && len(scanEntries) != len(*beamIJs) {
		return []ScanPoint{}, fmt.Errorf(`ScanEntry length %v doesn't match image beam location length %v for image: %v`, len(scanEntries), len(*beamIJs), imageName)
	}

	// Loop through all the scan entries and build what we need
	firstBeam := true

	for c := 0; c < len(scanEntries); c++ {
		scanEntry := scanEntries[c]

		beamXYZ := beamLocations[c]
		var imageIJ *protos.Coordinate2D
		if beamIJs != nil {
			imageIJ = (*beamIJs)[c]
		}
		var imageIJPoint *Point
		if scanEntry.Location && beamXYZ != nil && (imageIJ != nil || beamIJs == nil) {
			if beamIJs == nil {
				imageIJPoint = &Point{float64(beamXYZ.X), float64(beamXYZ.Y)}
			} else {
				if readBeamIJSwapped {
					// backwards (the old, buggy way it was for 5 years after project inception)
					imageIJPoint = &Point{float64(imageIJ.I), float64(imageIJ.J)}
				} else {
					// i=row (aka y), j=col (aka x)
					imageIJPoint = &Point{float64(imageIJ.J), float64(imageIJ.I)}
				}
			}

			// Expand the x,y,z bbox:
			locPointXMinMax.expand(float64(beamXYZ.X))
			locPointYMinMax.expand(float64(beamXYZ.Y))
			locPointZMinMax.expand(float64(beamXYZ.Z))

			if beamIJs != nil && imageIJ != nil {
				// And the i,j bbox
				// Not sure why this was rounded in past, but keeping this convention going forward until
				// a need arises to change it
				pixlX := imageIJ.J
				pixlY := imageIJ.I

				if readBeamIJSwapped {
					// backwards (the old, buggy way it was for 5 years after project inception)
					pixlX = imageIJ.I
					pixlY = imageIJ.J
				}

				roundedIJ := convertLocationComponentToPixelPosition(pixlX, pixlY)
				if firstBeam {
					g.locationPointBBox = Rect{roundedIJ.X, roundedIJ.Y, 0, 0}
					firstBeam = false
				} else {
					g.locationPointBBox.expandToFitPoint(roundedIJ)
				}
			}

			g.locationCount++
		}

		scanPt := ScanPoint{
			uint32(scanEntry.Id),
			imageIJPoint,
			int32(c),
			scanEntry.NormalSpectra > 0,
			scanEntry.DwellSpectra > 0,
			scanEntry.PseudoIntensities,
			scanEntry.PseudoIntensities && scanEntry.NormalSpectra == 0,
		}
		scanPoints = append(scanPoints, scanPt)

		if scanEntry.DwellSpectra > 0 || scanEntry.NormalSpectra > 0 {
			g.locationsWithNormalSpectra++
		}
	}

	if g.locationCount <= 0 {
		return []ScanPoint{}, errors.New("No location information found")
	}

	fmt.Printf(`  Location position relative to context image: (x,y)=%v,%v, (w,h)=%v,%v
`, g.locationPointBBox.X, g.locationPointBBox.Y, g.locationPointBBox.W, g.locationPointBBox.H)

	// store sizing
	g.locationPointXSize = locPointXMinMax.getRange()
	g.locationPointYSize = locPointYMinMax.getRange()
	g.locationPointZSize = locPointZMinMax.getRange()

	g.locationPointZMax = *locPointZMinMax.Max

	fmt.Printf(`  Location data physical size X=%v, Y=%v, Z=%v
`, g.locationPointXSize, g.locationPointYSize, g.locationPointZSize)
	return scanPoints, nil
}

func convertLocationComponentToPixelPosition(x float32, y float32) Point {
	return Point{math.Round(float64(x)), math.Round(float64(y))}
}

/* Example mismatch: Polygon 78 in 214827527

UI poly:
_Point {x: 356.66934535179854, y: 329.0211977576498}
_Point {x: 359.4916058411135, y: 328.67925887021966}
_Point {x: 359.51126609973596, y: 330.0440661473042} <-- extra point generated
_Point {x: 359.52045041687984, y: 330.681637763922}
_Point {x: 356.7454157761925, y: 330.73726411015747}
_Point {x: 356.6791169221013, y: 329.79581908243364}

API poly:
{i: 356.6693420410156, j: 329.0212097167969}
{i: 359.4916076660156, j: 328.67926025390625}
{i: 359.52044677734375, j: 330.681640625}
{i: 356.74542236328125, j: 330.7372741699219}
{i: 356.6791076660156, j: 329.7958068847656}
*/

func makeScanPointPolygons(bboxExpand float64, cluster PointCluster, scanPoints []ScanPoint, scanPointPolygons []ScanPointPolygon) error {
	// Create a larger bbox to ensure all polygons generated extend past the hull
	var clusterBBox *Rect
	sites := []vornoi.SiteVertex{}

	for _, locIdx := range cluster.LocIdxs {
		loc := scanPoints[locIdx]

		if loc.coord != nil && (loc.hasNormalSpectra || loc.hasPseudoIntensities) {
			// normal spectra may not be down yet!
			pt := vornoi.SiteVertex{Vertex: vornoi.Vertex{X: loc.coord.X, Y: loc.coord.Y}, Data: locIdx}
			if clusterBBox == nil {
				clusterBBox = &Rect{pt.X, pt.Y, 0, 0}
			} else {
				clusterBBox.expandToFitPoint(*loc.coord)
			}

			sites = append(sites, pt)
		}
	}

	if clusterBBox == nil {
		// haven't found valid points
		fmt.Printf("WARNING: No valid points for generating PMC polygons for: [%v]\n", cluster.LocIdxs)
		return nil
	}

	bbox := vornoi.NewBBox((*clusterBBox).X-bboxExpand, clusterBBox.maxX()+bboxExpand, clusterBBox.Y-bboxExpand, clusterBBox.maxY()+bboxExpand) // xl is x-left, xr is x-right, yt is y-top, and yb is y-bottom

	// Compute diagram and close cells (add half edges from bounding box)
	diagram := vornoi.ComputeDiagram(sites, bbox, true)

	hullPoly := makePolygolGeomFromHull(cluster.FootprintPoints)

	for c, cell := range diagram.Cells {
		if len(cell.Halfedges) > 2 {
			v := cell.Halfedges[0].GetStartpoint()

			polyPts := []Point{{v.X, v.Y}}

			for _, e := range cell.Halfedges {
				v = e.GetEndpoint()
				polyPts = append(polyPts, Point{v.X, v.Y})
			}

			siteLocIdx := -1
			if v, ok := cell.Site.Data.(int); !ok {
				return fmt.Errorf("Vornoi cell %v had invalid Data set: %v", c, cell.Site.Data)
			} else {
				siteLocIdx = v
			}

			clipPoly := makePolygolGeom(polyPts)

			// Clip polygon against the hull
			hullClippedPolyPts, err := polygol.Intersection(clipPoly, hullPoly)
			if err != nil {
				return fmt.Errorf("Failed to clip cluster polygon %v to hull: %v", c, err)
			}

			// Also against the biggest polygon we want to allow
			angle, err := getAngleForLocation(siteLocIdx, cluster.AngleRadiansToContextImage, scanPoints)
			if err != nil {
				return fmt.Errorf("Failed to get polygon angle for: %v", c)
			}

			polyPts, err = clipAgainstLargestPolyAllowed(
				hullClippedPolyPts,
				scanPoints[siteLocIdx],
				(cluster.PointDistance/2)*1.25,
				angle,
			)
			if err != nil {
				return fmt.Errorf("Failed to clip cluster polygon %v: %v", c, err)
			}

			// Now we convert it back to Points
			if siteLocIdx >= len(scanPointPolygons) {
				return fmt.Errorf("Vornoi cell %v doesn't have a corresponding polygon", c)
			}

			// Snip off last one, it's a repeat of first point
			if len(polyPts) > 0 {
				scanPointPolygons[siteLocIdx].Points = polyPts[0 : len(polyPts)-1]
				scanPointPolygons[siteLocIdx].updateBBox()
			}
		}
	}

	return nil
}

func makePolygolGeom(poly []Point) polygol.Geom {
	vals := [][]float64{}

	for _, pt := range poly {
		vals = append(vals, []float64{pt.X, pt.Y})
	}

	return polygol.Geom{{vals}}
}

func makePolygolGeomFromHull(poly []HullPoint) polygol.Geom {
	vals := [][]float64{}

	for _, pt := range poly {
		vals = append(vals, []float64{pt.Point.X, pt.Point.Y})
	}

	return polygol.Geom{{vals}}
}

func clipAgainstLargestPolyAllowed(polyPts polygol.Geom, loc ScanPoint, maxBoxSize float64, clusterAngleRad float64) ([]Point, error) {
	// Generate the largest allowable polygon for the point in question
	if loc.coord == nil {
		return []Point{}, nil
	}

	boxes := makeRotatedBoxes([]Point{*loc.coord}, maxBoxSize, clusterAngleRad)
	if len(boxes) != 1 && len(boxes[0]) != 4 {
		return []Point{}, nil
	}

	clipBox := makePolygolGeom(boxes[0])

	// Do the clip
	clipped, err := polygol.Intersection(polyPts, clipBox)
	if err != nil {
		return []Point{}, err
	}

	result := []Point{}
	for _, pts := range clipped[0][0] {
		result = append(result, Point{pts[0], pts[1]})
	}
	return result, nil
}

func getAngleForLocation(locIdx int, clusterAngleRad float64, scanPoints []ScanPoint) (float64, error) {
	// Get the 2 points around it. If this isn't possible just use the cluster angle
	if locIdx <= 0 || locIdx >= len(scanPoints)-1 {
		return clusterAngleRad, nil
	}

	pt := scanPoints[locIdx].coord

	preIdx := locIdx - 1
	postIdx := locIdx + 1

	// If they have a coordinate...
	prePt := scanPoints[preIdx].coord
	postPt := scanPoints[postIdx].coord

	if pt == nil || prePt == nil || postPt == nil {
		return clusterAngleRad, nil
	}

	// If somehow we ended up with the same points, we can't generate an angle here...
	// Found this issue with Baker Springs test dataset (from breadboard)
	if vectorsEqual(*prePt, *pt) || vectorsEqual(*pt, *postPt) {
		fmt.Printf("WARNING: Found equivalent PMC coordinates, failed to generate angle. PMCs around: %v\n", scanPoints[locIdx].PMC)
		return clusterAngleRad, nil
	}

	// Find the vectors, add them
	preVecN := normalizeVector(getVectorBetweenPoints(*prePt, *pt))
	postVecN := normalizeVector(getVectorBetweenPoints(*pt, *postPt))

	// If the angle between them is > 60 degrees...
	angleAroundPt := math.Acos(getVectorDotProduct(preVecN, postVecN))
	if angleAroundPt > math.Pi/3 {
		// We assume we're at a turning point and we'll just use the overall angle
		return clusterAngleRad, nil
	}

	vecN := normalizeVector(addVectors(preVecN, postVecN))

	// Get its angle to axis
	compareAxis := Point{0, -1}
	if vecN.X < 0 {
		compareAxis.Y = 1
	}

	result := math.Acos(getVectorDotProduct(compareAxis, vecN))

	if math.IsInf(result, 0) || math.IsNaN(result) {
		return 0, errors.New("NaN in getAngleForLocation")
	}

	return result, nil
}

/*
func RunGeneratePolygons(scanId string, imageNameForLocations string, imageBeamVersion uint8, svcs *services.APIServices) {

}
*/
