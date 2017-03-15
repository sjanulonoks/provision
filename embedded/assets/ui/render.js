/* jshint esversion: 6 */

var SubnetForm = React.createClass({
  getInitialState() {
    return this.getDefaultState();
  },

  getDefaultState() {
    return {
      Name: "",
      Subnet: "",
      NextServer: "",
      ActiveStart: "",
      ActiveEnd: "",
      ActiveLeaseTime: 0,
      ReservedLeaseTime: 7200,
      OnlyReservations: true,
      Options: {
        "3": "",
        "6": "",
        "15": "",
        "67": ""
      },
      "Strategy": "MAC"
    };
  },

  handleChange(event) {
    this.setState({[event.target.name]: event.target.value});
  },

  handleSubmit() {

  },

  shouldComponentUpdate(nextProps, nextState) {
    this.setState(nextProps.value);
    return true;
  },

  render() {
    var inputs = [
      {name: "Name", type: "text", display: "Name", placeholder: "eth0"},
      {name: "Subnet", type: "text", display: "Subnet", placeholder: "10.10.10.1/24"},
      {name: "NextServer", type: "text", display: "Next Server", placeholder: "10.10.10.1"},
      {name: "ActiveStart", type: "text", display: "Active Start", placeholder: "10.10.10.0"},
      {name: "ActiveEnd", type: "text", display: "Active End", placeholder: "10.10.10.255"},
      {name: "ActiveLeaseTime", type: "text", display: "Active Lease Time", placeholder: "0"},
      {name: "ReservedLeaseTime", type: "text", display: "Reserved Lease Time", placeholder: "7200"},
      {name: "OnlyReservations", type: "checkbox", display: "Only Reservations", placeholder: "true"}
    ];

    return (
      <form onSubmit={this.handleSubmit}>
        <table className="input-table">
          <tbody>
            {inputs.map(obj =>
              <tr key={obj.name}>
                <td>{obj.display}:</td>
                <td>
                  <input
                    type={obj.type}
                    name={obj.name}
                    placeholder={obj.placeholder}
                    value={this.state[obj.name]}
                    onChange={this.handleChange}/>
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </form>
    );
  }
})

var Subnets = React.createClass({
  getInitialState() {
    return {
      subnets: [],
      interfaces: [],
      subnetVal: {}
    };
  },
  
  getSubnets() {
    return new Promise((resolve, reject) => {
      var subnets = {};
      var interfaces = {};

      $.getJSON("../api/v3/interfaces", data => {
        for(var key in data) {
          var iface = data[key];
          iface.broadcast = true;
          interfaces[iface.Name] = iface;
        }

        // add interfaces to subnets list
        $.getJSON("../api/v3/subnets", data => {
          for(var key in data) {
            var subnet = data[key];
            subnet.broadcast = typeof interfaces[subnet.Name] !== 'undefined';
            delete interfaces[subnet.Name];
            subnets[subnet.Name] = subnet;
          }

          resolve({
            interfaces: Object.keys(interfaces).map(k => interfaces[k]),
            subnets: Object.keys(subnets).map(k => subnets[k])
          });
        });
      });
    });
  },

  componentDidMount() {
    this.getSubnets().then(data => {
      this.setState({
        subnets: data.subnets,
        interfaces: data.interfaces
      });
    });
  },

  updateForm(data) {
    return () => {
      this.setState({subnetVal: data});
    };
  },

  render() {
    return (
    <div>
      <table className="fullwidth">
        <thead>
          <tr>
            <th>Name/NIC</th>
            <th>Type</th>
            <th>Subnet</th>
            <th>Reservations</th>
            <th>Active Lease Time</th>
            <th>Reserved Lease Time</th>
            <th>Range</th>
          </tr>
        </thead>
        <tbody>{this.state.subnets.map(
          (val) => (<tr key={val.Name}>
            <td><a onClick={this.updateForm(val)}>{val.Name}</a></td>
            <td>{val.broadcast ? "broadcast" : "relayed"}</td>
            <td>{val.Subnet}</td>
            <td>{val.OnlyReservations ? "required" : "optional"}</td>
            <td>{val.ActiveLeaseTime}</td>
            <td>{val.ReservedLeaseTime}</td>
            <td>{val.ActiveStart} to {val.ActiveEnd}</td>
          </tr>)
        )}</tbody>
      </table>

      <SubnetForm value={this.state.subnetVal}/>

      <h2 style={{display: "flex", flexDirection: "row", alignItems: "center"}}>
        <span>Available Interfaces: </span>
        <span className="interface-list">
          {this.state.interfaces.map(val => 
            val.Addresses.map(subval =>
              <a key={val.Name} className="interface-pair">
                <header>{val.Name}</header>
                <subhead>{subval}</subhead>
              </a>
            )
          )}
        </span>
      </h2>
    </div>
    );
  }
});

ReactDOM.render(<Subnets />, subnets)
