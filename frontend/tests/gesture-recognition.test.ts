import { describe, expect, test } from "bun:test"
import cases from "../../testdata/gesture-recognition.json"
import { directionsFromPoints } from "../src/lib/gesture-recognition"

for (const pixelRatio of [1, 1.25, 1.5, 2, 2.5, 3, 4]) {
  describe(`屏幕缩放 ${pixelRatio * 100}%，与实际触发共用回归用例`, () => {
    for (const fixture of cases) {
      test(fixture.name, () => {
        const points = fixture.points.map(([x, y]) => ({ x: x / pixelRatio, y: y / pixelRatio }))
        expect(directionsFromPoints(points, fixture.distance, pixelRatio)).toEqual(fixture.want)
      })
    }
  })
}
