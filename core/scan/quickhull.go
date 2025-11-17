package scan

import "errors"

type HullPoint struct {
	Point  Point
	Idx    int
	Normal *Point
}

type QuickHullGenerator struct {
	hull []HullPoint
}

func QuickHull(points []HullPoint) ([]HullPoint, error) {
	if len(points) < 3 {
		return []HullPoint{}, errors.New("Convex hull not possible, not enough points")
	}

	minPt, maxPt := getMinMaxPoints(points)

	qh := QuickHullGenerator{}
	qh.addSegments(minPt, maxPt, points)
	qh.addSegments(maxPt, minPt, points) //reverse line direction to get points on other side
	return qh.hull, nil
}

/**
 * Return the min and max points in the set along the X axis
 * Returns [ {x,y}, {x,y} ]
 * @param {Array} points - An array of {x,y} objects
 */
func getMinMaxPoints(points []HullPoint) (HullPoint, HullPoint) {
	minPoint := points[0]
	maxPoint := points[0]

	//for(i=1; i<points.length; i++) {
	for _, pt := range points {
		if pt.Point.X < minPoint.Point.X {
			minPoint = pt
		}
		if pt.Point.X > maxPoint.Point.X {
			maxPoint = pt
		}
	}

	return minPoint, maxPoint
}

/**
 * Calculates the distance of a point from a line
 * @param {Array} point - Array [x,y]
 * @param {Array} line - Array of two points [ [x1,y1], [x2,y2] ]
 */
func distanceFromLine(point HullPoint, line1 HullPoint, line2 HullPoint) float64 {
	vY := line2.Point.Y - line1.Point.Y
	vX := line1.Point.X - line2.Point.X
	return (vX*(point.Point.Y-line1.Point.Y) + vY*(point.Point.X-line1.Point.X))
}

/**
 * Determines the set of points that lay outside the line (positive), and the most distal point
 * Returns: {points: [ [x1, y1], ... ], max: [x,y] ]
 * @param points
 * @param line
 */
func distalPoints(line1 HullPoint, line2 HullPoint, points []HullPoint) ([]HullPoint, *HullPoint) {
	outer_points := []HullPoint{}
	var distal_point *HullPoint
	var distance float64
	var max_distance float64

	for _, point := range points {
		distance = distanceFromLine(point, line1, line2)

		if distance > 0 {
			outer_points = append(outer_points, point)
		} else {
			continue //short circuit
		}

		if distance > max_distance {
			distal_point = &point
			max_distance = distance
		}
	}

	return outer_points, distal_point
}

// Recursively adds hull segments
func (q *QuickHullGenerator) addSegments(line1 HullPoint, line2 HullPoint, points []HullPoint) {
	outerPoints, maxPt := distalPoints(line1, line2, points)

	if maxPt == nil {
		q.hull = append(q.hull, line1)
		return
	}

	q.addSegments(line1, *maxPt, outerPoints)
	q.addSegments(*maxPt, line2, outerPoints)
}
