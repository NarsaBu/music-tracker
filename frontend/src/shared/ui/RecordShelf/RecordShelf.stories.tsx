import type { Meta, StoryObj } from '@storybook/react-vite';
import { RecordSleeve } from '../RecordSleeve/RecordSleeve';
import { TapeLabel } from '../TapeLabel/TapeLabel';
import { RecordShelf } from './RecordShelf';

const meta = { title: 'UI/RecordShelf', component: RecordShelf } satisfies Meta<typeof RecordShelf>;
export default meta;
type Story = StoryObj<typeof meta>;

export const Shelf: Story = {
  args: {
    label: 'Релизы',
    children: ['Ágætis byrjun', 'Von', '( )', 'Takk…', 'Kveikur'].map((title) => (
      <RecordSleeve
        key={title}
        title={title}
        subtitle="Sigur Rós"
        meta="1999"
        badges={<TapeLabel>Album</TapeLabel>}
      />
    )),
  },
};
