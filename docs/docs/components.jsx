import React from 'react';
import Card from '@mui/material/Card';
import CardActions from '@mui/material/CardActions';
import CardHeader from '@mui/material/CardHeader';
import CardContent from '@mui/material/CardContent';
import Button from '@mui/material/Button';
import LayersIcon from '@mui/icons-material/Layers';
import TerminalIcon from '@mui/icons-material/Terminal';
import Grid from '@mui/material/Grid';
import CloudQueueIcon from '@mui/icons-material/CloudQueue';
import { ThemeProvider, createTheme } from '@mui/material/styles';
import { useColorMode } from '@docusaurus/theme-common'

export default function PrimaryUseCases() {
    const { isDarkTheme } = useColorMode();

    const darkTheme = createTheme({
      palette: {
        mode: isDarkTheme ? 'dark' : 'light',
      }
    })

    return (
      <ThemeProvider theme={darkTheme}>
        <Grid container spacing={4}>

        <Grid item xs={12} s={12} md={4}>
            <Card style={{ height: '100%' }}>
                <CardHeader title="API Gateway" avatar={<CloudQueueIcon />} titleTypographyProps={{variant:'h6'}} />
                <CardContent>                    
                    Build REST web services with HTTP handling, caching, OAuth, and much more.
                </CardContent>
                <CardActions>
                    <Button size="small" href="/gosoline/category/api-gateway">Get started</Button>
                </CardActions>
            </Card>
        </Grid>

        <Grid item xs={12} md={4}>
            <Card style={{ height: '100%' }}>
                <CardHeader title="Message Queues" avatar={<LayersIcon />} titleTypographyProps={{variant:'h6'}} />
                <CardContent>
                    Process asynchronous messages from Kafka, Redis, or any other queuing or streaming system.
                </CardContent>
                <CardActions>
                    <Button size="small" href="/gosoline/quickstart/create-a-consumer">Get started</Button>
                </CardActions>
            </Card>
        </Grid>

        <Grid item xs={12} md={4}>
            <Card style={{ height: '100%' }}>
                <CardHeader title="Kernel Module" avatar={<TerminalIcon />} titleTypographyProps={{variant:'h6'}} />
                <CardContent>                                    
                    Implement a kernel module with which you can do anything, using gosoline's logging, configuration, and other solutions.
                </CardContent>
                <CardActions>
                    <Button size="small" href="/gosoline/quickstart/create-an-application">Get started</Button>
                </CardActions>
            </Card>
        </Grid>

        </Grid>
      </ThemeProvider>
    )
  }