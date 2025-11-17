package scan

import "math"

type Point struct {
	x float64
	y float64
}

func getVectorBetweenPoints(pt1 Point, pt2 Point) Point {
	return Point{pt2.x - pt1.x, pt2.y - pt1.y}
}

func getVectorDotProduct(v1 Point, v2 Point) float64 {
	return v1.x*v2.x + v1.y*v2.y
}

func addVectors(v1 Point, v2 Point) Point {
	return Point{v1.x + v2.x, v1.y + v2.y}
}
func subtractVectors(v1 Point, v2 Point) Point {
	return Point{v1.x - v2.x, v1.y - v2.y}
}

func getVectorLength(v Point) float64 {
	return math.Sqrt(float64(v.x*v.x + v.y*v.y))
}

func normalizeVector(v Point) Point {
	len := getVectorLength(v)
	return scaleVector(v, 1/len)
}

func vectorsEqual(v1 Point, v2 Point) bool {
	return v1.x == v2.x && v1.y == v2.y
}

func scaleVector(v Point, s float64) Point {
	return Point{v.x * s, v.y * s}
}

func getRotationMatrix(angleRad float64) [][]float64 {
	return [][]float64{
		{math.Cos(angleRad), -math.Sin(angleRad), 0},
		{math.Sin(angleRad), math.Cos(angleRad), 0},
		{0, 0, 1},
	}
}

func pointByMatrix(m [][]float64, v Point) Point {
	return Point{
		m[0][0]*v.x + m[1][0]*v.y + m[2][0],
		m[0][1]*v.x + m[1][1]*v.y + m[2][1],
	}
}

type ScanPointPolygon struct {
	bbox   Rect
	points []Point
}

func (p *ScanPointPolygon) updateBBox() {
	if len(p.points) > 0 {
		for c, pt := range p.points {
			if c == 0 {
				p.bbox = Rect{pt.x, pt.y, 0, 0}
			} else {
				p.bbox.expandToFitPoint(pt)
			}
		}
	}
}

type Rect struct {
	x float64
	y float64
	w float64
	h float64
}

func (r *Rect) maxX() float64 {
	return r.x + r.w
}

func (r *Rect) maxY() float64 {
	return r.y + r.h
}

func (r *Rect) expandToFitPoint(pt Point) {
	if pt.x < r.x {
		r.w += r.x - pt.x
		r.x = pt.x
	}

	if pt.y < r.y {
		r.h += r.y - pt.y
		r.y = pt.y
	}

	tmp := r.maxX()
	if pt.x > tmp {
		r.w += pt.x - tmp
	}
	tmp = r.maxY()
	if pt.y > tmp {
		r.h += pt.y - tmp
	}
}

/*
  private _bbox: Rect = new Rect(0, 0, 0, 0);

  constructor(public points: Point[]) {
    this.updateBBox();
  }

  updateBBox() {
    if (this.points.length > 0) {
      this._bbox = new Rect(this.points[0].x, this.points[0].y, 0, 0);
      this._bbox.expandToFitPoints(this.points);
    }
  }

  get bbox(): Rect {
    return this._bbox;
  }
}*/
