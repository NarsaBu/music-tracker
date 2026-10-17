import type { Meta, StoryObj } from '@storybook/react-vite';
import { Led } from '../Led/Led';
import { PushButton } from '../PushButton/PushButton';
import { IndexCard } from './IndexCard';

const meta = {
  title: 'UI/IndexCard',
  component: IndexCard,
  decorators: [(Story) => <div style={{ maxWidth: 320 }}>{Story()}</div>],
} satisfies Meta<typeof IndexCard>;
export default meta;
type Story = StoryObj<typeof meta>;

export const Artist: Story = {
  args: {
    title: 'Sigur Rós',
    tab: 'IS',
    tiltSeed: 'f6f2326f-6b25-4170-b89d-e235b25508e8',
    extra: <Led state="ok" label="Синхронизирован" />,
    children: <span>Icelandic post‐rock band · Group</span>,
    footer: <PushButton>Отписаться</PushButton>,
  },
};

export const Minimal: Story = { args: { title: 'Nirvana' } };

export const LongName: Story = {
  args: {
    title: 'Godspeed You! Black Emperor & A Silver Mt. Zion Memorial Orchestra',
    tab: 'CA',
    tiltSeed: 'long',
  },
};
