<template>
  <div id="app">

    <header>
      <b-container fluid class="px-0" id="nav">
        <b-navbar toggleable="sm" type="dark">
          <b-navbar-brand>Enduro</b-navbar-brand>
          <b-navbar-toggle target="nav-collapse"></b-navbar-toggle>
          <b-collapse id="nav-collapse" is-nav>
            <b-navbar-nav>
              <b-nav-item :to="{ name: 'collections' }">Collections</b-nav-item>
            </b-navbar-nav>
            <b-navbar-nav class="ml-auto">
              <b-nav-item href="https://enduroproject.netlify.com">Docs</b-nav-item>
              <b-nav-item href="https://github.com/artefactual-labs/enduro">GitHub</b-nav-item>
            </b-navbar-nav>
          </b-collapse>
        </b-navbar>
      </b-container>
    </header>

    <router-view/>

    <en-footer/>

  </div>
</template>

<style lang="scss">

@import 'node_modules/bootstrap/scss/bootstrap';
@import 'node_modules/bootstrap-vue/src/index.scss';

#app {
  font-family: 'Avenir', Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  color: #2c3e50;
}

#nav {
  .navbar-brand {
    user-select: none;
  }
  nav {
    background-color: $enduro-c1 !important;
  }
  @media (min-width: 576px) {
    .nav-link.router-link-active {
      color: white;
      background-color: #fff2;
    }
  }
  .nav-link.router-link-active {
    @extend .rounded-sm;
  }
}

footer {
  font-size: .8rem;
  .container {
    text-align: center;
    border-top: 1px solid $enduro-bolight;
    color: #999;
  }
}

</style>
