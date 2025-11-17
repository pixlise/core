package scan

import "math"

type Point struct {
	X float64
	Y float64
}

func getVectorBetweenPoints(pt1 Point, pt2 Point) Point {
	return Point{pt2.X - pt1.X, pt2.Y - pt1.Y}
}

func getVectorDotProduct(v1 Point, v2 Point) float64 {
	return v1.X*v2.X + v1.Y*v2.Y
}

func addVectors(v1 Point, v2 Point) Point {
	return Point{v1.X + v2.X, v1.Y + v2.Y}
}
func subtractVectors(v1 Point, v2 Point) Point {
	return Point{v1.X - v2.X, v1.Y - v2.Y}
}

func getVectorLength(v Point) float64 {
	return math.Sqrt(float64(v.X*v.X + v.Y*v.Y))
}

func normalizeVector(v Point) Point {
	len := getVectorLength(v)
	return scaleVector(v, 1/len)
}

func vectorsEqual(v1 Point, v2 Point) bool {
	return v1.X == v2.X && v1.Y == v2.Y
}

func scaleVector(v Point, s float64) Point {
	return Point{v.X * s, v.Y * s}
}

func getRotationMatrix(angleRad float64) [][]float64 {
	return [][]float64{
		{math.Cos(angleRad), -math.Sin(angleRad), 0},
		{math.Sin(angleRad), math.Cos(angleRad), 0},
		{0, 0, 1},
	}
}

func pointByMatrix(m [][]float64, v Point) Point {
	pt := Point{
		m[0][0]*v.X + m[0][1]*v.Y + m[0][2],
		m[1][0]*v.X + m[1][1]*v.Y + m[1][2],
	}
	w := m[2][0]*v.X + m[2][1]*v.Y + m[2][2]

	if w == 1 || w == 0 {
		return pt
	}
	return Point{pt.X / w, pt.Y / w}
}

type ScanPointPolygon struct {
	bbox   Rect
	points []Point
}

func (p *ScanPointPolygon) updateBBox() {
	if len(p.points) > 0 {
		for c, pt := range p.points {
			if c == 0 {
				p.bbox = Rect{pt.X, pt.Y, 0, 0}
			} else {
				p.bbox.expandToFitPoint(pt)
			}
		}
	}
}

type Rect struct {
	X float64
	Y float64
	W float64
	H float64
}

func (r *Rect) maxX() float64 {
	return r.X + r.W
}

func (r *Rect) maxY() float64 {
	return r.Y + r.H
}

func (r *Rect) expandToFitPoint(pt Point) {
	if pt.X < r.X {
		r.W += r.X - pt.X
		r.X = pt.X
	}

	if pt.Y < r.Y {
		r.H += r.Y - pt.Y
		r.Y = pt.Y
	}

	tmp := r.maxX()
	if pt.X > tmp {
		r.W += pt.X - tmp
	}
	tmp = r.maxY()
	if pt.Y > tmp {
		r.H += pt.Y - tmp
	}
}

/*
  private _bbox: Rect = new Rect(0, 0, 0, 0);

  constructor(public points: Point[]) {
    this.updateBBox();
  }

  updateBBox() {
    if (this.points.length > 0) {
      this._bbox = new Rect(this.points[0].X, this.points[0].Y, 0, 0);
      this._bbox.expandToFitPoints(this.points);
    }
  }

  get bbox(): Rect {
    return this._bbox;
  }
}*/
