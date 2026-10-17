/**
 * Spread of the marks around the knob, degrees from 12 o'clock. From four
 * options on, a narrower arc puts the upper labels on top of each other.
 */
function arcFor(count: number): number {
  return count <= 3 ? 90 : 180;
}

/** Angle of option i out of count, evenly spread over the arc. */
export function angleOf(i: number, count: number): number {
  if (count <= 1) return 0;
  const arc = arcFor(count);
  return -arc / 2 + (arc / (count - 1)) * i;
}
