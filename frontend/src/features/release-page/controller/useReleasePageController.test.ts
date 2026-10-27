import { renderHook, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { createWrapper } from '@/test/queryWrapper';
import { server } from '@/test/server';
import { useReleasePageController } from './useReleasePageController';

const OKC = 'b1392450-e666-3926-a536-22c65f834433';

const detail = {
  mbid: OKC,
  title: 'OK Computer',
  artist_credit: 'Radiohead',
  primary_type: 'album',
  secondary_types: [],
  first_release_date: '1997-05-21',
  is_announcement: false,
  cover_url: `https://coverartarchive.org/release-group/${OKC}/front-250`,
  musicbrainz_url: `https://musicbrainz.org/release-group/${OKC}`,
  rating: { value: 4.55, votes: 89 },
};

function renderController() {
  const { Wrapper } = createWrapper();
  return renderHook(() => useReleasePageController(OKC), { wrapper: Wrapper });
}

describe('useReleasePageController', () => {
  it('shows both ratings, reviews and search links', async () => {
    server.use(
      http.get(`/api/v1/release-groups/${OKC}`, () => HttpResponse.json(detail)),
      http.get(`/api/v1/release-groups/${OKC}/reviews`, () =>
        HttpResponse.json({
          items: [
            {
              id: 'r1',
              author: 'Jon Lusk',
              rating: 4,
              text: 'Fine.',
              language: 'en',
              published_at: '2007-04-25T00:00:00Z',
              license: 'CC BY-SA 3.0',
              url: 'https://critiquebrainz.org/review/r1',
            },
          ],
          total: 3,
          average: { value: 4, votes: 1 },
          url: `https://critiquebrainz.org/release-group/${OKC}`,
        }),
      ),
    );
    const { result } = renderController();

    await waitFor(() => expect(result.current.title).toBe('OK Computer'));
    expect(result.current.details).toBe('Альбом');
    expect(result.current.releaseDate).toBe('21 мая 1997');
    expect(result.current.musicbrainz).toEqual({
      valueText: '4,6 / 5',
      votesText: 'голосов: 89',
      percent: 91,
    });
    expect(result.current.links?.albumOfTheYear).toBe(
      'https://www.albumoftheyear.org/search/?q=Radiohead%20OK%20Computer',
    );

    await waitFor(() => expect(result.current.reviews).toHaveLength(1));
    expect(result.current.reviews[0]).toMatchObject({ author: 'Jon Lusk', stars: 4 });
    expect(result.current.critiquebrainz.valueText).toBe('4,0 / 5');
    // Three reviews, one shown: link to the rest.
    expect(result.current.allReviewsUrl).toBe(`https://critiquebrainz.org/release-group/${OKC}`);
  });

  it('shows genres, the tracklist and links to credited artists', async () => {
    const RADIOHEAD = 'a74b1b7f-71a5-4011-9441-d0b5e4122711';
    const GUEST = 'f4fdbb4c-e4b7-47a0-b83b-d91bbfcfa387';
    server.use(
      http.get(`/api/v1/release-groups/${OKC}`, () => HttpResponse.json(detail)),
      http.get(`/api/v1/release-groups/${OKC}/details`, () =>
        HttpResponse.json({
          genres: ['alternative rock', 'art rock'],
          edition: {
            mbid: 'e1',
            title: 'OK Computer',
            date: '1997-05-21',
            country: 'JP',
            status: 'Official',
            musicbrainz_url: 'https://musicbrainz.org/release/e1',
          },
          media: [
            {
              position: 1,
              format: 'CD',
              title: '',
              tracks: [
                { number: '1', title: 'Airbag', length_ms: 284000, artist_credit: 'Radiohead' },
                {
                  number: '2',
                  title: 'Duet',
                  length_ms: null,
                  artist_credit: 'Radiohead feat. Guest',
                },
              ],
            },
          ],
          artists: [{ mbid: RADIOHEAD, name: 'Radiohead', subscribed: true }],
          guests: [{ mbid: GUEST, name: 'Guest', subscribed: false }],
        }),
      ),
    );
    const { result } = renderController();

    await waitFor(() => expect(result.current.media).toHaveLength(1));
    expect(result.current.genres).toEqual(['alternative rock', 'art rock']);
    expect(result.current.artists).toEqual([
      { mbid: RADIOHEAD, name: 'Radiohead', external: false, to: `/artists/${RADIOHEAD}` },
    ]);
    expect(result.current.guests).toEqual([
      {
        mbid: GUEST,
        name: 'Guest',
        external: true,
        href: `https://musicbrainz.org/artist/${GUEST}`,
      },
    ]);
    // A single unnamed medium needs no heading; only the guest track shows its credit.
    expect(result.current.media[0]).toMatchObject({
      heading: null,
      tracks: [
        { number: '1', title: 'Airbag', length: '4:44', credit: null },
        { number: '2', title: 'Duet', length: '', credit: 'Radiohead feat. Guest' },
      ],
    });
    expect(result.current.tracklistSummary).toBe('Треков: 2 · 4:44');
    expect(result.current.editionText).toBe('По изданию: OK Computer · JP · 21 мая 1997');
  });

  it('keeps the page when reviews are down and has no votes yet', async () => {
    server.use(
      http.get(`/api/v1/release-groups/${OKC}`, () =>
        HttpResponse.json({ ...detail, rating: null }),
      ),
      http.get(`/api/v1/release-groups/${OKC}/reviews`, () =>
        HttpResponse.json(
          { type: 'about:blank', title: 'Upstream unavailable', status: 503 },
          { status: 503, headers: { 'Content-Type': 'application/problem+json' } },
        ),
      ),
    );
    const { result } = renderController();

    await waitFor(() => expect(result.current.reviewsError).toBeDefined());
    expect(result.current.reviewsError).toBe('CritiqueBrainz не отвечает, попробуйте позже');
    expect(result.current.title).toBe('OK Computer');
    expect(result.current.musicbrainz).toMatchObject({
      valueText: null,
      votesText: 'ещё нет оценок',
    });
  });

  it('tells a release the tracker does not know', async () => {
    server.use(
      http.get(`/api/v1/release-groups/${OKC}`, () =>
        HttpResponse.json(
          { type: 'about:blank', title: 'Not found', status: 404 },
          { status: 404, headers: { 'Content-Type': 'application/problem+json' } },
        ),
      ),
      http.get(`/api/v1/release-groups/${OKC}/reviews`, () =>
        HttpResponse.json({}, { status: 404 }),
      ),
    );
    const { result } = renderController();
    await waitFor(() => expect(result.current.notFound).toBe(true));
    expect(result.current.loadFailed).toBe(false);
  });
});
