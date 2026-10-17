import type { Meta, StoryObj } from '@storybook/react-vite';
import { CassetteDropzone } from './CassetteDropzone';

const meta = {
  title: 'UI/CassetteDropzone',
  component: CassetteDropzone,
  args: {
    label: 'Кассетоприёмник',
    hint: '.txt или .csv, до 1 МБ',
    accept: '.txt,.csv',
    onFile: () => {},
  },
} satisfies Meta<typeof CassetteDropzone>;
export default meta;
type Story = StoryObj<typeof meta>;

export const Empty: Story = {};
export const Playing: Story = { args: { tapeLabel: 'artists.csv', busy: true } };
export const Disabled: Story = { args: { disabled: true } };
