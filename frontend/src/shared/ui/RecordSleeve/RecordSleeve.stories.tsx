import type { Meta, StoryObj } from '@storybook/react-vite';
import { TapeLabel } from '../TapeLabel/TapeLabel';
import { RecordSleeve } from './RecordSleeve';

const meta = {
  title: 'UI/RecordSleeve',
  component: RecordSleeve,
  decorators: [
    (Story) => (
      <div style={{ width: 180, padding: 16, background: 'var(--surface-room)' }}>{Story()}</div>
    ),
  ],
} satisfies Meta<typeof RecordSleeve>;
export default meta;
type Story = StoryObj<typeof meta>;

export const WithCover: Story = {
  args: {
    title: 'Ágætis byrjun',
    subtitle: 'Sigur Rós',
    coverUrl:
      'https://coverartarchive.org/release-group/62b427b8-1c52-34e7-b250-da2aa3e44860/front-250',
    meta: 'июн 1999',
    badges: <TapeLabel>Album</TapeLabel>,
    href: 'https://musicbrainz.org/release-group/62b427b8-1c52-34e7-b250-da2aa3e44860',
  },
};

export const NoCover: Story = {
  args: {
    title: 'Untitled Announcement',
    subtitle: 'Sigur Rós feat. Someone',
    meta: 'дата неизвестна',
    badges: (
      <>
        <TapeLabel>Single</TapeLabel>
        <TapeLabel tone="accent">Анонс</TapeLabel>
      </>
    ),
  },
};

export const Muted: Story = { args: { ...NoCover.args, muted: true } };
