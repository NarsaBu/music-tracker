import type { Meta, StoryObj } from '@storybook/react-vite';
import { Led } from '../Led/Led';
import { Panel } from './Panel';

const meta = { title: 'UI/Panel', component: Panel } satisfies Meta<typeof Panel>;
export default meta;
type Story = StoryObj<typeof meta>;

export const WithTitle: Story = {
  args: { title: 'Release filter', children: 'Содержимое панели' },
};

export const WithExtra: Story = {
  args: {
    title: 'Email digest',
    extra: <Led state="ok" label="Включено" />,
    children: 'Содержимое панели',
  },
};
