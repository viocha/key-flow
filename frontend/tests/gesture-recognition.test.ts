import { describe, expect, test } from "bun:test"
import cases from "../../testdata/gesture-recognition.json"
import { directionsFromPoints } from "../src/lib/gesture-recognition"

describe("与实际触发共用的逻辑像素回归用例", () => {
  for (const fixture of cases) {
    test(fixture.name, () => {
      const points = fixture.points.map(([x, y]) => ({ x, y }))
      expect(directionsFromPoints(points, fixture.distance)).toEqual(fixture.want)
    })
  }
})
