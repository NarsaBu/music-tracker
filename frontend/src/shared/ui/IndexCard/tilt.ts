const MAX_TILT_DEG = 0.6;

/** Deterministic angle in [-0.6°, 0.6°] from a string (FNV-1a hash). */
export function tiltOf(seed: string): number {
  let h = 0x811c9dc5;
  for (let i = 0; i < seed.length; i++) {
    h ^= seed.charCodeAt(i);
    h = Math.imul(h, 0x01000193);
  }
  const unit = (h >>> 0) / 0xffffffff; // 0…1
  return Math.round((unit * 2 - 1) * MAX_TILT_DEG * 100) / 100;
}
